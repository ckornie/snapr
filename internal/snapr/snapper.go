package snapr

import (
	"context"
	"fmt"
	"snapr/internal/zed"
	"strconv"
	"strings"
	"time"
)

type snapper struct {
	ctx      context.Context
	zed      *zed.Zed
	settings *Settings
}

func (s *Snapr) newSnapper() snapper {
	return snapper{s.ctx, s.zed, s.settings}
}

func (s *snapper) entries() map[string][]SnapEntry {
	targets := make(map[string][]SnapEntry)
	for k, v := range s.settings.FileSystems {
		targets[k] = v.Snap
	}
	return targets
}

func (s snapper) Snap() []string {
	snapshots := make([]string, 0)

	for target, entries := range s.entries() {
		if len(entries) == 0 {
			Logger.Info().Msgf("skipping snapshot on '%s': no entries", target)
			continue
		}

		fs, err := zed.ToFileSystem(target)
		if err != nil {
			Logger.Warn().Msgf("skipping snapshot on '%s': failed parsing file system (%s)", target, err)
			continue
		}

		for _, entry := range entries {
			if snapshot, err := s.snap(*fs, entry); err != nil {
				Logger.Warn().Msgf("failed creating snapshot %s on '%s': %s", fs, entry.Prefix, err)
			} else {
				if snapshot != nil {
					Logger.Info().Msgf("created snapshot '%s' on '%s'", snapshot.Address(), target)
					snapshots = append(snapshots, snapshot.Address())
				}
			}
		}
	}
	return snapshots
}

func (s snapper) snap(fs zed.FileSystem, entry SnapEntry) (*zed.Snapshot, error) {
	interval, err := entry.IntervalDuration()
	if err != nil {
		return nil, fmt.Errorf("could not parse interval '%s': %w", entry.Interval, err)
	}

	listing, err := s.zed.ListSnapshots(s.ctx, fs)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots for '%s': %w", fs, err)
	}

	if s.expired(entry.Prefix, interval, listing) {
		snapshot := nextSnap(fs, entry.Prefix, listing)
		if err = s.zed.CreateSnapshot(s.ctx, snapshot); err != nil {
			return nil, fmt.Errorf("failed to create snapshot '%s': %w", snapshot.Address(), err)
		}

		for _, tag := range entry.Hold {
			if err = s.zed.HoldSnapshot(s.ctx, snapshot, tag); err != nil {
				Logger.Warn().Err(err).Stack().Msgf("failed to apply hold '%s' to snapshot '%s'", tag, snapshot.Address())
			}
		}
		return &snapshot, nil
	}
	return nil, nil
}

func nextSnap(fs zed.FileSystem, prefix string, listing []zed.SnapshotListing) zed.Snapshot {
	token := prefix + "-"

	last := -1
	for _, v := range listing {
		if v.Snapshot.Addr.FileSystem == fs {
			split := strings.Split(v.Snapshot.Addr.Name, token)
			if len(split) > 1 {
				seq, err := strconv.Atoi(split[len(split)-1])
				if err == nil {
					if seq > last {
						last = seq
					}
				}
			}
		}
	}

	return zed.Snapshot{
		Addr: zed.Address{
			FileSystem: fs,
			Name:       fmt.Sprintf("%s%s", token, padNumber(last+1)),
		},
	}
}

func (s snapper) expired(prefix string, interval time.Duration, listing []zed.SnapshotListing) bool {
	token := prefix + "-"
	for _, v := range listing {
		if strings.HasPrefix(v.Snapshot.Addr.Name, token) {
			if v.Created.After(time.Now().Add(-interval)) {
				return false
			}
		}
	}
	return true
}
