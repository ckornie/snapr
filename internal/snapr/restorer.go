package snapr

import (
	"context"
	"fmt"
	"snapr/internal/zed"
)

type restorer struct {
	ctx      context.Context
	zed      *zed.Zed
	settings *Settings
}

func (s *Snapr) newRestorer() *restorer {
	return &restorer{
		ctx:      s.ctx,
		zed:      s.zed,
		settings: s.settings,
	}
}

func (r *restorer) restore(target string) error {
	entries, ok := r.settings.FileSystems[target]
	if !ok {
		Logger.Warn().Msgf("restore failed for %s: not configured", target)
	}

	if len(entries.Send) == 0 {
		return fmt.Errorf("restore failed for %s: at least one send entry required", target)
	}

	entry := entries.Send[len(entries.Send)-1].Inherit(r.settings)

	fs, err := zed.ToFileSystem(target)
	if err != nil {
		return fmt.Errorf("restore failed for %s: %w", target, err)
	}

	remote, err := newRemote(r.ctx, r.zed, entry)
	if err != nil {
		return fmt.Errorf("restore failed for %s: %w", target, err)
	}

	if err := remote.restore(*fs); err != nil {
		return fmt.Errorf("restore failed for %s: %w", target, err)
	}

	Logger.Info().Msgf("restored %s", target)
	return nil
}
