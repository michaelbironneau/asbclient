package asbclient

import (
	"fmt"
	"testing"
)

type sasPiece struct {
	Namespace  string
	SAKeyName  string
	SAKeyValue string
	QueuePath  string
	URI        string
	Expiry     string
	ToSign     string
	Signature  string
	Auth       string
}

//testPieces are fragments collected by print()ing the output of the Python client at various stages.
var testPieces = map[string]sasPiece{
	"tester": {
		Namespace:  "tester",
		SAKeyName:  "RootManageSharedAccessKey",
		SAKeyValue: "gC9nJzD3UoxDP8LvQWkQihlvb6dBHpdxh7hXj3Trk5s=",
		QueuePath:  "stuff",
		URI:        "https%3a%2f%2ftester.servicebus.windows.net%3a443%2fstuff%2fmessages%2fhead%3ftimeout%3d60",
		Expiry:     "1461590684",
		ToSign:     "https%3a%2f%2ftester.servicebus.windows.net%3a443%2fstuff%2fmessages%2fhead%3ftimeout%3d60\n1461590684",
		Signature:  "Efyus8Vg3NUBCIY3qxvqvGVnaK6cYlcEqaTRctzB%2B04%3D",
		Auth:       "SharedAccessSignature sig=Efyus8Vg3NUBCIY3qxvqvGVnaK6cYlcEqaTRctzB%2B04%3D&se=1461590684&skn=RootManageSharedAccessKey&sr=https%3a%2f%2ftester.servicebus.windows.net%3a443%2fstuff%2fmessages%2fhead%3ftimeout%3d60",
	},
}

func TestSAS(t *testing.T) {
	var tu string
	for testName, testPieces := range testPieces {
		aq := New(Queue, testPieces.Namespace, testPieces.SAKeyName, testPieces.SAKeyValue)
		testRequestURI := fmt.Sprintf(serviceBusURL+ testPieces.QueuePath + "/messages/head?timeout=60", aq.namespace)

		if tu = aq.signatureURI(testRequestURI); tu != testPieces.URI {
			t.Errorf("%s expected uri \n %s \n but got \n %s \n", testName, testPieces.URI, tu)
		}

		if ts := aq.stringToSign(tu, testPieces.Expiry); ts != testPieces.ToSign {
			t.Errorf("%s expected uri \n %s \n but got \n %s \n", testName, testPieces.ToSign, ts)
		}

		if te := aq.authHeader(testRequestURI, testPieces.Expiry); te != testPieces.Auth {
			t.Errorf("%s expected signature \n %s \n but got \n %s \n", testName, testPieces.Auth, te)
		}
	}
}
