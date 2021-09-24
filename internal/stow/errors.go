package stow

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func errorFromResponse(res http.Response) error {
	defer res.Body.Close()
	if b, err := ioutil.ReadAll(res.Body); err == nil {
		return newStatusError("status code", res, b)
	}
	return newStatusError("status code", res, nil)
}

func newStatusError(description string, res http.Response, body []byte) error {
	return &StatusError{
		Description: description,
		StatusCode:  res.StatusCode,
		Status:      res.Status,
		Message:     string(body),
	}
}

// StatusError represents an error occurring with a stow operation.
type StatusError struct {
	Description string
	StatusCode  int
	Status      string
	Message     string
}

func (e *StatusError) Error() string {
	if e.StatusCode < 200 && e.StatusCode >= 300 {
		if len(e.Status) > 0 {
			return fmt.Sprintf("%s: status code %d\n%s", e.Description, e.StatusCode, e.Status)
		}
		if len(e.Message) > 0 {
			return fmt.Sprintf("%s: status code %d\n%s", e.Description, e.StatusCode, e.Message)
		}
		return fmt.Sprintf("%s: status code %d", e.Description, e.StatusCode)
	}
	if len(e.Message) > 0 {
		return fmt.Sprintf("%s:\n%s", e.Description, e.Message)
	}
	return fmt.Sprintf("%s: unknown cause", e.Description)
}
