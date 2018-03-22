package asbclient

import (
	"fmt"
	"strings"
	"time"
)

//Message is an Azure Service Bus message
type MessageReq struct {
  ContentType string `json:",omitempty"`
  CorrelationId string `json:",omitempty"`
  SessionID string `json:"SessionId,omitempty"`
  Label string `json:",omitempty"`
  ReplyTo string `json:",omitempty"`
  // TODO: Time span...
  // TimeToLive string `json:",omitempty"`
  To string `json:",omitempty"`
  ScheduledEnqueueTimeUtc Time `json:",omitempty"`
  ReplyToSessionId string `json:",omitempty"`
  PartitionKey string `json:",omitempty"`

  Body []byte `json:"-"`
}

type Message struct {
  MessageReq
  DeliveryCount          int
	EnqueuedSequenceNumber int
	EnqueuedTimeUtc        Time
	LockToken              string
	LockedUntilUtc         Time
	MessageID              string `json:"MessageId"`
	SequenceNumber         int
	State                  string
	Location string

  Body []byte `json:"-"`

}

// Time is a wrapper round time.Time to json encode/decode in RFC1123 fomat
type Time struct {
	time.Time
}

// UnmarshalJSON from RFC1123
func (t *Time) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		t.Time = time.Time{}
		return
	}
	t.Time, err = time.Parse(time.RFC1123, s)
	return
}

var nilTime = (time.Time{}).UnixNano()

// MarshalJSON to RFC1123
func (t *Time) MarshalJSON() ([]byte, error) {
	if t.Time.UnixNano() == nilTime {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", t.Time.Format(time.RFC1123))), nil
}
