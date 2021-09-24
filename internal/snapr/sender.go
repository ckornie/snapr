package snapr

import (
	"context"
	"snapr/internal/zed"
)

type sender struct {
	ctx      context.Context
	zed      *zed.Zed
	settings *Settings
}

func (s *Snapr) newSender() *sender {
	return &sender{
		s.ctx,
		s.zed,
		s.settings,
	}
}

func (s *sender) entries() map[string][]SendEntry {
	targets := make(map[string][]SendEntry)
	for k, v := range s.settings.FileSystems {
		targets[k] = v.Send
	}
	return targets
}

// Send performs uploads as per configuration.
func (s *sender) Send() {
	for target, entries := range s.entries() {
		if len(entries) == 0 {
			Logger.Warn().Msgf("sending failed for %s: no sends", target)
			continue
		}

		fs, err := zed.ToFileSystem(target)
		if err != nil {
			Logger.Warn().Msgf("sending failed for %s: %s", target, err)
			continue
		}

		for _, entry := range entries {
			remote, err := newRemote(s.ctx, s.zed, entry.Inherit(s.settings))
			if err != nil {
				Logger.Warn().Msgf("sending failed for %s: %s", target, err)
				continue
			}

			if err := remote.refresh(*fs); err != nil {
				Logger.Warn().Msgf("sending failed for %s: %s", target, err)
			}
		}
	}
}
