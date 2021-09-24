package zed

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Bookmark represents a bookmark of a snapshot.
type Bookmark struct {
	Addr   Address
	Exists bool
}

// Address returns a way to reference the bookmark.
func (b Bookmark) Address() string {
	return b.Addr.asBookmark()
}

// BookmarkListing represents a bookmark with associated meta-data.
type BookmarkListing struct {
	Bookmark Bookmark
	Created  time.Time
}

// ListBookmarks lists all bookmarks for a file system.
func (z *Zed) ListBookmarks(ctx context.Context, fs FileSystem) ([]BookmarkListing, error) {
	cmd := exec.CommandContext(ctx, z.path, "list", "-H", "-r", "-t", "bookmark", "-o", "name,creation", fs.String())

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	listing := make([]BookmarkListing, 0)

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		row := scanner.Text()
		fields := strings.SplitN(row, "\t", 2)
		if len(fields) != 2 {
			return nil, fmt.Errorf("list bookmarks failed: error parsing row '%s'", row)
		}

		addr, err := NewAddress(fields[0], "#")
		if err != nil {
			return nil, err
		}

		creation, err := time.ParseInLocation(creationTime, fields[1], time.Local)
		if err != nil {
			return nil, err
		}

		listing = append(listing, BookmarkListing{Bookmark{*addr, true}, creation.UTC()})
	}
	return listing, nil
}

// CreateBookmark creates a bookmark of a snapshot.
func (z *Zed) CreateBookmark(ctx context.Context, bookmark Bookmark, source Snapshot) (*Bookmark, error) {
	cmd := exec.CommandContext(ctx, z.path, "bookmark", source.Address(), bookmark.Address())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to bookmark '%s': %s (%w)", bookmark.Address(), parseError(out), err)
	}
	return &bookmark, nil
}
