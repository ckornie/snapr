package snapr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"snapr/internal/stow"
	"time"
)

// SnapArguments holds options for running snap.
type SnapArguments struct {
	Active bool
}

// SendArguments holds options for running send.
type SendArguments struct {
	Active bool
}

// RestoreArguments holds options for running restore.
type RestoreArguments struct {
	Active bool
}

// SendEntry holds options for running restore.
type SendEntry struct {
	Endpoint   string
	Region     string
	Account    string
	Secret     string
	Bucket     string
	Release    []string
	Threads    int
	VolumeSize int
	PartSize   int
}

// SnapEntry holds options for a snapshot schedule.
type SnapEntry struct {
	Interval string
	Prefix   string
	Hold     []string
}

// IntervalDuration will retrieve the interval as a duration.
func (e SnapEntry) IntervalDuration() (time.Duration, error) {
	return time.ParseDuration(e.Interval)
}

// Validate will perform checks on arguments.
func (e SendEntry) Validate() error {
	if e.Endpoint == "" {
		return fmt.Errorf("missing endpoint")
	}

	if e.Region == "" {
		return fmt.Errorf("missing region")
	}

	if e.Account == "" {
		return fmt.Errorf("missing account")
	}

	if e.Secret == "" {
		return fmt.Errorf("missing secret")
	}

	if e.Bucket == "" {
		return fmt.Errorf("missing bucket name")
	}
	return nil
}

// Inherit will inherit unset values from the parent.
func (e SendEntry) Inherit(settings *Settings) SendEntry {
	if e.Threads == 0 {
		e.Threads = settings.Threads
	}
	if e.VolumeSize == 0 {
		e.VolumeSize = settings.VolumeSize
	}
	if e.PartSize == 0 {
		e.PartSize = settings.PartSize
	}
	return e
}

// NewStow creates a stow instance from the stored fields.
func (e SendEntry) NewStow() (*stow.Stow, error) {
	settings, err := stow.NewSettings(
		stow.Use(e.Endpoint, e.Region),
		stow.WithCredentials(e.Account, e.Secret),
		stow.WithLogger(Logger),
	)
	if err != nil {
		return nil, err
	}
	return stow.New(settings)
}

// Settings represents the configuration.
type Settings struct {
	FileSystems map[string]FileSystemSettings
	Threads     int
	VolumeSize  int
	PartSize    int
}

// FileSystemSettings represent per file system settings.
type FileSystemSettings struct {
	Send []SendEntry
	Snap []SnapEntry
}

// NewSettings instantiates new settings with default values.
func NewSettings() *Settings {
	return &Settings{
		Threads:    Threads,
		VolumeSize: VolumeSize,
		PartSize:   PartSize,
	}
}

// Load will load settings from a JSON encoded configuration file.
func (s *Settings) Load(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("unable to load settings from %s (%w)", file, err)
	}

	defer f.Close()

	raw, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("unable to read settings from %s (%w)", file, err)
	}

	err = json.Unmarshal(raw, s)
	if err != nil {
		return fmt.Errorf("unable to unmarshal settings from %s (%w)", file, err)
	}
	return nil
}
