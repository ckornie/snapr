package stow

import (
	"net/http"
	"time"
)

// Forwarder provides a way to perform HTTP operations.
type Forwarder func(*http.Request) (*http.Response, error)

// NewForwarder creates an instance using http.Client.
func NewForwarder(pool int) Forwarder {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxConnsPerHost = pool + 1
	t.MaxIdleConnsPerHost = pool + 1
	c := http.Client{
		Transport: t,
		Timeout:   10 * time.Minute,
	}

	return func(req *http.Request) (*http.Response, error) {
		res, err := c.Do(req)

		if err == nil && res.StatusCode >= 200 && res.StatusCode < 300 {
			return res, nil
		}

		if res != nil {
			return nil, errorFromResponse(*res)
		}
		return nil, err
	}
}
