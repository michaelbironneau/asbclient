package asbclient

import (
	"fmt"
	"strings"
	"time"
)

//Message is an Azure Service Bus message
type Message struct {
	DeliveryCount          int
	EnqueuedSequenceNumber int
	EnqueuedTimeUtc        Time
	LockToken              string
	LockedUntilUtc         Time
	MessageID              string `json:"MessageId"`
	PartitionKey           string
	SequenceNumber         int
	State                  string
	TimeToLive             int

	Location string

	Body []byte
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
