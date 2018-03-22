//Package asbclient is a client for Azure Service Bus.
package asbclient

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

//ErrSubscriptionRequired is raised when a service bus topics operations is performed without a subscription.
var ErrSubscriptionRequired = fmt.Errorf("A subscription is required for Service Bus Topic operations")

//ClientType denotes the type of client (topic/queue) that is required
type ClientType int

const (
	//Queue is a client type for Azure Service Bus queues
	Queue ClientType = iota

	//Topic is a client type for Azure Service Bus topics
	Topic ClientType = iota
)

//Client is a client for Azure Service Bus (queues and topics). You should use a different client instance for every namespace.
//For more comprehensive documentation on its various methods, see:
//	https://msdn.microsoft.com/en-us/library/azure/hh780717.aspx
type Client interface {
	DeleteMessage(item *Message) error
	PeekLockMessage(path string, timeout int) (*Message, error)
	Send(path string, item *MessageReq) error
	SetSubscription(subscription string)
	Unlock(item *Message) error
}

//client is the default implementation of Client
type client struct {
	clientType   ClientType
	namespace    string
	subscription string
	saKey        string
	saValue      []byte
	url          string
	client       *http.Client
}

const serviceBusURL = "https://%s.servicebus.windows.net:443/"
const apiVersion = "2017-04"

//New creates a new client from the given parameters. Their meaning can be found in the MSDN docs at:
//  https://docs.microsoft.com/en-us/rest/api/servicebus/Introduction
func newClient(clientType ClientType, namespace string, sharedAccessKeyName string, sharedAccessKeyValue string) *client {
	return &client{
		clientType: clientType,
		namespace:  namespace,
		saKey:      sharedAccessKeyName,
		saValue:    []byte(sharedAccessKeyValue),
		url:        fmt.Sprintf(serviceBusURL, namespace),
		client:     &http.Client{},
	}
}

//New creates a new client from the given parameters. Their meaning can be found in the MSDN docs at:
//  https://docs.microsoft.com/en-us/rest/api/servicebus/Introduction
func New(clientType ClientType, namespace string, sharedAccessKeyName string, sharedAccessKeyValue string) Client {
	return newClient(clientType, namespace, sharedAccessKeyName, sharedAccessKeyValue)
}

//SetSubscription sets the client's subscription. Only required for Azure Service Bus Topics.
func (c *client) SetSubscription(subscription string) {
	c.subscription = subscription
}

func (c *client) request(url string, method string) (*http.Request, error) {
	return c.requestWithBody(url, method, &MessageReq{})
}

func (c *client) requestWithBody(urlString string, method string, message *MessageReq) (*http.Request, error) {

	url, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	q := url.Query()
	q.Set("api-version", apiVersion)
	url.RawQuery = q.Encode()

	req, err := http.NewRequest(method, url.String(), bytes.NewBuffer(message.Body)) // TODO: handle existing query params
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authHeader(url.String(), c.signatureExpiry(time.Now())))
  props, _ := json.Marshal(message)
  req.Header.Set("brokerProperties", string(props))

	return req, nil
}

//DeleteMessage deletes the message.
//
//For more information see https://docs.microsoft.com/en-us/rest/api/servicebus/delete-message.
func (c *client) DeleteMessage(item *Message) error {

	req, err := c.request(item.Location, "DELETE")

	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)

	if err != nil {
		return err
	}

	io.Copy(ioutil.Discard, resp.Body)

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return readError(resp)
}

//Send sends a new item to `path`, where `path` is either the queue name or the topic name.
//
//For more information see https://docs.microsoft.com/en-us/rest/api/servicebus/send-message-to-queue.
func (c *client) Send(path string, item *MessageReq) error {
	req, err := c.requestWithBody(c.url+path+"/messages/", "POST", item)

	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		io.Copy(ioutil.Discard, resp.Body)
		return nil
	}

	return readError(resp)
}

//Unlock unlocks a message for processing by other receivers.
//
//For more information see https://docs.microsoft.com/en-us/rest/api/servicebus/unlock-message.
func (c *client) Unlock(item *Message) error {
	req, err := c.request(item.Location+"/"+item.LockToken, "PUT")

	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		io.Copy(ioutil.Discard, resp.Body)
		return nil
	}

	return readError(resp)
}

//PeekLockMessage atomically retrieves and locks the latest message from the queue or topic at `path` (which should not include slashes).
//
//If using this with a service bus topic, make sure you SetSubscription() first.
//For more information see https://docs.microsoft.com/en-us/rest/api/servicebus/peek-lock-message-non-destructive-read.
func (c *client) PeekLockMessage(path string, timeout int) (*Message, error) {
	var url string
	if c.clientType == Queue {
		url = c.url + path + "/"
	} else {
		if c.subscription == "" {
			return nil, ErrSubscriptionRequired
		}
		url = c.url + path + "/subscriptions/" + c.subscription + "/"
	}
	req, err := c.request(url+fmt.Sprintf("messages/head?timeout=%d", timeout), "POST")

	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNoContent {
		io.Copy(ioutil.Discard, resp.Body)
		return nil, nil
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, readError(resp)
	}

	defer resp.Body.Close()
	mBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading message body")
	}

	brokerProperties := resp.Header.Get("BrokerProperties")

	location := resp.Header.Get("Location")

	var message Message

	if err := json.Unmarshal([]byte(brokerProperties), &message); err != nil {
		return nil, fmt.Errorf("Error unmarshalling BrokerProperties: %v", err)
	}

	message.Location = location
	message.Body = mBody

	return &message, nil
}

//signatureExpiry returns the expiry for the shared access signature for the next request.
//
//It's translated from the Python client:
// https://github.com/Azure/azure-sdk-for-python/blob/master/azure-servicebus/azure/servicebus/servicebusservice.py
func (c *client) signatureExpiry(from time.Time) string {
	t := from.Add(300 * time.Second).Round(time.Second).Unix()
	return strconv.Itoa(int(t))
}

//signatureURI returns the canonical URI according to Azure specs.
//
//It's translated from the Python client:
//https://github.com/Azure/azure-sdk-for-python/blob/master/azure-servicebus/azure/servicebus/servicebusservice.py
func (c *client) signatureURI(uri string) string {
	return strings.ToLower(url.QueryEscape(uri)) //Python's urllib.quote and Go's url.QueryEscape behave differently. This might work, or it might not...like everything else to do with authentication in Azure.
}

//stringToSign returns the string to sign.
//
//It's translated from the Python client:
//https://github.com/Azure/azure-sdk-for-python/blob/master/azure-servicebus/azure/servicebus/servicebusservice.py
func (c *client) stringToSign(uri string, expiry string) string {
	return uri + "\n" + expiry
}

//signString returns the HMAC signed string.
//
//It's translated from the Python client:
//https://github.com/Azure/azure-sdk-for-python/blob/master/azure-servicebus/azure/servicebus/_common_conversion.py
func (c *client) signString(s string) string {
	h := hmac.New(sha256.New, c.saValue)
	h.Write([]byte(s))
	encodedSig := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return url.QueryEscape(encodedSig)
}

//authHeader returns the value of the Authorization header for requests to Azure Service Bus.
//
//It's translated from the Python client:
//https://github.com/Azure/azure-sdk-for-python/blob/master/azure-servicebus/azure/servicebus/servicebusservice.py
func (c *client) authHeader(uri string, expiry string) string {
	u := c.signatureURI(uri)
	s := c.stringToSign(u, expiry)
	sig := c.signString(s)
	return fmt.Sprintf("SharedAccessSignature sig=%s&se=%s&skn=%s&sr=%s", sig, expiry, c.saKey, u)
}
