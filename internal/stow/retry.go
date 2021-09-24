package stow

import (
	"errors"
	"time"
)

type retryStrategy interface {
	retry(cause error) time.Duration
}

type fixedRetry struct {
	count     int
	delay     time.Duration
	retryable statusCodes
	fatal     []error
}

type statusCodes []int

func (s statusCodes) contains(statusCode int) bool {
	for _, code := range s {
		if statusCode == code {
			return true
		}
	}
	return false
}

func (r fixedRetry) retry(cause error) time.Duration {
	for _, e := range r.fatal {
		if errors.Is(cause, e) {
			return 0
		}
	}

	var res *StatusError
	if errors.As(cause, &res) {
		if !r.retryable.contains(res.StatusCode) {
			return 0
		}
	}

	r.count = r.count - 1
	if r.count >= 0 {
		return r.delay
	}
	return 0
}
