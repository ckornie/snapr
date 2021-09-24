package zed

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Snapshot represents a snapshot.
type Snapshot struct {
	Addr Address
}

// Address returns a way to reference the snapshot.
func (s Snapshot) Address() string {
	return s.Addr.asSnapshot()
}

// SnapshotListing represents a listed snapshot.
type SnapshotListing struct {
	Snapshot    Snapshot
	Created     time.Time
	Identity    string
	Transaction int
	Holds       []string
}

// ListSnapshots lists all snapshots for a target.
func (z *Zed) ListSnapshots(ctx context.Context, target FileSystem) ([]SnapshotListing, error) {
	cmd := exec.CommandContext(ctx, z.path, "list", "-H", "-r", "-t", "snapshot", "-o", "name,creation,guid,createtxg", "-s", "createtxg", target.String())

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	listing := make([]SnapshotListing, 0)

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		row := scanner.Text()
		fields := strings.SplitN(row, "\t", 4)
		if len(fields) != 4 {
			return nil, fmt.Errorf("list snapshots failed: error parsing row '%s'", row)
		}

		addr, err := NewAddress(fields[0], "@")
		if err != nil {
			return nil, err
		}

		creation, err := time.ParseInLocation(creationTime, fields[1], time.Local)
		if err != nil {
			return nil, err
		}

		transaction, err := strconv.Atoi(fields[3])
		if err != nil {
			return nil, err
		}

		snapshot := Snapshot{*addr}
		holds, err := z.ListHolds(ctx, snapshot)
		if err != nil {
			return nil, err
		}

		listing = append(listing, SnapshotListing{snapshot, creation.UTC(), fields[2], transaction, holds})
	}
	return listing, nil
}

// CreateSnapshot creates a snapshot.
func (z *Zed) CreateSnapshot(ctx context.Context, snapshot Snapshot) error {
	cmd := exec.CommandContext(ctx, z.path, "snapshot", snapshot.Address())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to snapshot '%s': %s (%w)", snapshot.Address(), parseError(out), err)
	}
	return nil
}

// HoldSnapshot places a hold on a snapshot.
func (z *Zed) HoldSnapshot(ctx context.Context, snapshot Snapshot, tag string) error {
	cmd := exec.CommandContext(ctx, z.path, "hold", tag, snapshot.Address())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to hold '%s': %s (%w)", tag, parseError(out), err)
	}
	return nil
}

// ReleaseSnapshot releases a hold on a snapshot.
func (z *Zed) ReleaseSnapshot(ctx context.Context, snapshot Snapshot, tag string) error {
	cmd := exec.CommandContext(ctx, z.path, "release", tag, snapshot.Address())
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.HasSuffix(string(out), "no such tag on this dataset") {
			return nil
		}
		return fmt.Errorf("failed to release '%s': %s (%w)", tag, parseError(out), err)
	}
	return nil
}

// ListHolds will return any holds on a snapshot.
func (z *Zed) ListHolds(ctx context.Context, snapshot Snapshot) ([]string, error) {
	cmd := exec.CommandContext(ctx, z.path, "holds", "-H", snapshot.Address())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list holds: %s (%w)", parseError(out), err)
	}

	holds := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		row := scanner.Text()
		fields := strings.Split(row, "\t")
		if len(fields) != 3 {
			return nil, fmt.Errorf("list holds failed: error parsing row '%s'", row)
		}

		holds = append(holds, fields[1])
	}
	return holds, nil
}
