package asbclient

import (
	"time"
)

//Message is an Azure Service Bus message
type Message struct {
	ID string 
	Location string
	LockToken string
	DeliveryCount int 
	EnqueuedTimeUtc time.Time 
	LockedUntil time.Time
	Body []byte
}