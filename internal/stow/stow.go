package stow

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type requestFactory func(context.Context, string, string, []byte) (*http.Request, error)

type operation interface {
	formRequest(request requestFactory, p Provider) (*http.Request, error)
}

type get struct {
	ctx context.Context
}

func (g get) formRequest(request requestFactory, p Provider) (*http.Request, error) {
	return request(g.ctx, http.MethodGet, p.url(), nil)
}

// Metadata represents metadata returned in a typical response.
type Metadata struct {
	Reference string
	Request   string
}

func newMetadata(res *http.Response) Metadata {
	return Metadata{
		Reference: res.Header.Get("X-Amz-Id-2"),
		Request:   res.Header.Get("X-Amz-Request-Id"),
	}
}

// Stow exposes common S3 operations by utilizing the S3 REST API.
type Stow struct {
	log           zerolog.Logger
	provider      Provider
	forwarder     Forwarder
	retry         func() retryStrategy
	authenticator authenticator
}

// New returns an initialized Client based on the settings.
func New(s *Settings) (*Stow, error) {
	return &Stow{
		s.Log,
		s.Provider,
		s.Forwarder,
		s.Retry,
		newAuthenticator(s.Provider.Region, s.Credentials.Account, s.Credentials.Secret),
	}, nil
}

func (s *Stow) String() string {
	return fmt.Sprintf("%s", s.provider.Endpoint)
}

func (s *Stow) doOperation(o operation) (*http.Response, error) {
	var b []byte
	retries := s.retry()

	factory := func(ctx context.Context, method string, url string, body []byte) (*http.Request, error) {
		if len(body) > 0 {
			b = body
			return http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
		}
		return http.NewRequestWithContext(ctx, method, url, http.NoBody)
	}

	for {
		req, err := o.formRequest(factory, s.provider)
		if err != nil {
			return nil, err
		}

		if len(b) > 0 {
			if err = s.authenticator.withBody(req, b); err != nil {
				return nil, err
			}
		} else {
			s.authenticator.noBody(req)
		}

		res, err := s.forwarder(req)
		if err == nil {
			return res, nil
		}

		if delay := retries.retry(err); delay > 0 {
			s.log.Warn().Err(err).Stack().Msg("retrying request")
			time.Sleep(delay)
		} else {
			return nil, err
		}
	}
}
