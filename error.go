package asbclient

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Error is the xml structure returned for errors
type Error struct {
	Code   int
	Detail string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s (Code: %d)", e.Detail, e.Code)
}

func readError(resp *http.Response) error {
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)

	if len(b) == 0 {
		return fmt.Errorf("returned code: %d", resp.StatusCode)
	}

	var e Error
	err := xml.Unmarshal(b, &e)
	if err != nil {
		return fmt.Errorf("%s (failed to parse error as xml: %s)", string(b), err)
	}
	return &e
}
