package snapr

import (
	"snapr/internal/zed"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSnapperNext(t *testing.T) {
	fs := zed.FileSystem{Pool: "pool-0", Name: "test"}
	latest := time.Date(2021, time.October, 6, 21, 11, 0, 0, time.UTC)

	snapshots := []zed.SnapshotListing{
		{
			Snapshot: zed.Snapshot{Addr: zed.Address{FileSystem: fs, Name: "primary-"}},
			Created:  time.Date(2020, time.September, 2, 1, 57, 0, 0, time.UTC),
		},
		{
			Snapshot: zed.Snapshot{Addr: zed.Address{FileSystem: fs, Name: "out-of-band"}},
			Created:  time.Date(2020, time.September, 2, 1, 57, 0, 0, time.UTC),
		},
		{
			Snapshot: zed.Snapshot{Addr: zed.Address{FileSystem: fs, Name: "primary-00002"}},
			Created:  time.Date(2021, time.January, 5, 6, 6, 0, 0, time.UTC),
		},
		{
			Snapshot: zed.Snapshot{Addr: zed.Address{FileSystem: fs, Name: "primary-00003"}},
			Created:  latest,
		},
	}

	expected := zed.Snapshot{Addr: zed.Address{FileSystem: fs, Name: "primary-00004"}}
	actual := nextSnap(fs, "primary", snapshots)

	assert.Equal(t, expected, actual)
}
