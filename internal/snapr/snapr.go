package snapr

import (
	"context"
	"snapr/internal/zed"
)

// Snapr will snap, send, and restore ZFS file systems.
type Snapr struct {
	ctx      context.Context
	settings *Settings
	zed      *zed.Zed
}

// New instantiates a new instance of Snapr.
func New(ctx context.Context, settings *Settings) (*Snapr, error) {
	zed, err := zed.New()
	if err != nil {
		return nil, err
	}
	return &Snapr{ctx, settings, zed}, nil
}

// Snap creates snapshots according to the settings.
func (s *Snapr) Snap() {
	s.newSnapper().Snap()
}

// Send uploads a full or incremental stream based on the settings.
func (s *Snapr) Send() {
	s.newSender().Send()
}

// Restore restores a file system from a bucket.
func (s *Snapr) Restore(fileSystem string) error {
	return s.newRestorer().restore(fileSystem)
}
