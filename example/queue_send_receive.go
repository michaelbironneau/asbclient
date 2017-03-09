package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/michaelbironneau/asbclient"
)

func main() {
	log.Printf("Starting")

	client := asbclient.New(asbclient.Queue, os.Getenv("sb_namespace"), os.Getenv("sb_key_name"), os.Getenv("sb_key_value"))

	path := os.Getenv("sb_queue")

	go func() {
		i := 0
		for {

			log.Printf("Send: %d", i)
			err := client.Send(path, &asbclient.Message{
				Body: []byte(fmt.Sprintf("message %d", i)),
			})

			if err != nil {
				log.Printf("Send error: %s", err)
			} else {
				log.Printf("Sent: %d", i)
			}

			time.Sleep(time.Millisecond * 500)
			i++
		}
	}()

	for {
		log.Printf("Peeking...")
		msg, err := client.PeekLockMessage(path, 30)

		if err != nil {
			log.Printf("Peek error: %s", err)
		} else {
			log.Printf("Peeked message: '%s'", string(msg.Body))
			err = client.DeleteMessage(msg)
			if err != nil {
				log.Printf("Delete error: %s", err)
			} else {
				log.Printf("Deleted message")
			}
		}

		time.Sleep(time.Millisecond * 200)
	}

}
