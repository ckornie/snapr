package stow

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// SetOption allows overridding of settings.
type SetOption func(*Settings) error

// Credentials holds authentication details.
type Credentials struct {
	Account string
	Secret  string
}

// Provider represents an S3 compatible storage provider.
type Provider struct {
	Endpoint string
	Region   string
}

func (p Provider) urlBucket(name string) string {
	return fmt.Sprintf("https://%s.%s", name, p.Endpoint)
}

func (p Provider) url() string {
	return fmt.Sprintf("https://%s", p.Endpoint)
}

// Settings provide a set of options used to instantiate a client.
type Settings struct {
	Log         zerolog.Logger
	Provider    Provider
	Credentials Credentials
	Forwarder   Forwarder
	Retry       func() retryStrategy
}

// NewSettings creates a default settings and then applies any given overrides.
func NewSettings(overrides ...SetOption) (s *Settings, err error) {
	s = &Settings{
		Log: log.Logger,
	}

	for _, override := range overrides {
		if err := override(s); err != nil {
			return nil, err
		}
	}

	s.applyDefaults()

	if err := s.validate(); err != nil {
		return nil, err
	}
	return s, nil
}

// Clone makes a shallow copy so that any changes are not reflected in an instantiated client.
func (s *Settings) Clone() Settings {
	return Settings{
		Provider: Provider{
			Endpoint: s.Provider.Endpoint,
			Region:   s.Provider.Region,
		},
		Credentials: s.Credentials,
		Forwarder:   s.Forwarder,
		Retry:       s.Retry,
		Log:         s.Log,
	}
}

func (s *Settings) applyDefaults() {
	if s.Forwarder == nil {
		s.Forwarder = NewForwarder(10)
	}

	if s.Retry == nil {
		s.Retry = func() retryStrategy {
			return fixedRetry{
				count:     10,
				delay:     time.Second,
				retryable: []int{408, 429, 500, 502, 503, 504},
				fatal:     []error{context.Canceled},
			}
		}
	}
}

func (s *Settings) validate() error {
	if len(s.Provider.Endpoint) == 0 {
		return fmt.Errorf("an endpoint is required")
	}

	if len(s.Credentials.Account) == 0 {
		return fmt.Errorf("a key is required")
	}

	if len(s.Credentials.Secret) == 0 {
		return fmt.Errorf("a secret is required")
	}
	return nil
}

// WithCredentials sets the credentials used.
func WithCredentials(account, secret string) SetOption {
	return func(s *Settings) error {
		s.Credentials = Credentials{account, secret}
		return nil
	}
}

// Use sets the endpoint and region.
func Use(endpoint, region string) SetOption {
	return func(settings *Settings) error {
		settings.Provider = Provider{
			Endpoint: endpoint,
			Region:   region,
		}
		return nil
	}
}

// UseWasabi sets the endpoint for the region.
func UseWasabi(region string) SetOption {
	return func(settings *Settings) error {
		settings.Provider = Provider{
			Endpoint: fmt.Sprintf("s3.%s.wasabisys.com", region),
			Region:   region,
		}
		return nil
	}
}

// UseBackblaze sets the endpoint for the region.
func UseBackblaze(region string) SetOption {
	return func(settings *Settings) error {
		settings.Provider = Provider{
			Endpoint: fmt.Sprintf("s3.%s.backblazeb2.com", region),
			Region:   region,
		}
		return nil
	}
}

// WithLogger sets the logger used.
func WithLogger(log zerolog.Logger) SetOption {
	return func(settings *Settings) error {
		settings.Log = log
		return nil
	}
}
