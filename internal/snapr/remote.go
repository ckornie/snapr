package snapr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"snapr/internal/stow"
	"snapr/internal/zed"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"
)

var splitPath = regexp.MustCompile(`^(?P<fs>[^\/]*\/[^\/]*)\/(?P<archive>\d+)\/(?P<volume>\d+)$`)

type remote struct {
	ctx       context.Context
	zed       *zed.Zed
	stow      *stow.Stow
	entry     SendEntry
	catalogue catalogue
}

func newRemote(ctx context.Context, zed *zed.Zed, entry SendEntry) (*remote, error) {
	stow, err := entry.NewStow()
	if err != nil {
		return nil, err
	}

	listing, err := stow.ListAllKeys(ctx, entry.Bucket)
	if err != nil {
		return nil, err
	}

	catalogue := make(catalogue)
	catalogue.load(listing)

	return &remote{
		ctx:       ctx,
		zed:       zed,
		stow:      stow,
		entry:     entry,
		catalogue: catalogue,
	}, nil
}

func (r *remote) restore(fs zed.FileSystem) error {
	paths, err := r.catalogue.verify(fs.String())
	if err != nil {
		return err
	}

	Logger.Info().Msgf("restoring %s from %s", fs, r.entry.Bucket)

	for i, v := range paths {
		err := r.restoreVolume(fs, i, v)
		if err != nil {
			return fmt.Errorf("failed to restore %s (%w)", fs, err)
		}
	}
	return nil
}

func (r *remote) restoreVolume(fs zed.FileSystem, index int, volumes []string) error {
	out, in := io.Pipe()
	pool := fs.Pool

	eg, ctx := errgroup.WithContext(r.ctx)
	eg.Go(func() error {
		err := r.zed.Receive(ctx, pool, out)
		if err != nil {
			Logger.Warn().Msgf("restore failed for %s: volume %d failed", pool, index)
			cause := fmt.Errorf("%w", err)
			out.CloseWithError(err)
			return cause
		}

		Logger.Info().Msgf("volume %d has been restored to %s", index, pool)
		return nil
	})

	Logger.Debug().Msgf("restoring volume %d to %s", index, pool)

	for _, volume := range volumes {
		err := r.download(volume, in)
		if err != nil {
			in.CloseWithError(err)
			return err
		}
	}

	in.Close()
	return eg.Wait()
}

func (r *remote) download(path string, w io.Writer) error {
	position := 0
	offset := r.entry.PartSize*Megabyte - 1

	for i := 1; ; i++ {
		object, err := r.stow.GetObject(r.ctx, r.entry.Bucket, path, position, position+offset)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		n, err := w.Write(object.Content)
		if err != nil {
			return err
		}

		if object.End+1 == object.Size {
			Logger.Info().Msgf("downloaded %s (%d MB)", path, object.Size/Megabyte)
			return nil
		}

		Logger.Debug().Msgf("downloaded part %d (%d MB) of %s", i, n/Megabyte, path)
		position = object.End + 1
	}
}

func (r *remote) refresh(fs zed.FileSystem) error {
	listing, err := r.zed.ListSnapshots(r.ctx, fs)
	if err != nil {
		return err
	}

	archives, err := r.catalogue.verify(fs.String())
	if err != nil {
		return err
	}

	sequence := len(archives)
	if sequence > 0 {
		previous := fmt.Sprintf("%s/%s/contents", fs.String(), padNumber(sequence-1))
		contents, err := r.stow.GetObject(r.ctx, r.entry.Bucket, previous, 0, 0)
		if err != nil {
			return err
		}

		var entries []ArchiveEntry
		err = json.Unmarshal(contents.Content, &entries)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			return fmt.Errorf("no contents retained in %s", previous)
		}

		path := fmt.Sprintf("%s/%s", fs.String(), padNumber(sequence))
		return r.incremental(path, fs, listing, entries[len(entries)-1].Identity)
	}

	path := fmt.Sprintf("%s/%s", fs.String(), padNumber(sequence))
	return r.full(path, fs, listing)
}

func (r *remote) incremental(path string, fs zed.FileSystem, listing []zed.SnapshotListing, identity string) error {
	listing, err := r.zed.ListSnapshots(r.ctx, fs)
	if err != nil {
		return err
	}

	for i, v := range listing {
		if v.Identity == identity {
			if i == len(listing)-1 {
				return fmt.Errorf("remote is up to date")
			}
			target := listing[len(listing)-1]
			return r.send(path, &v.Snapshot, target.Snapshot, listing[i:])
		}
	}
	return fmt.Errorf("snapshot %s not found", identity)
}

func (r *remote) full(path string, fs zed.FileSystem, listing []zed.SnapshotListing) error {
	if len(listing) > 0 {
		target := listing[len(listing)-1]
		return r.send(path, nil, target.Snapshot, listing)
	}
	return fmt.Errorf("no snapshots exist")
}

func (r *remote) send(path string, source *zed.Snapshot, target zed.Snapshot, included []zed.SnapshotListing) error {
	completion := func(err error) error {
		if err == nil {
			for _, snapshot := range included[:len(included)-1] {
				for _, tag := range r.entry.Release {
					r.zed.ReleaseSnapshot(r.ctx, snapshot.Snapshot, tag)
				}
			}
		}
		return nil
	}

	stream, err := r.zed.Send(r.ctx, source, target, completion)
	if err != nil {
		return err
	}

	defer stream.Out.Close()

	upload, err := newUpload(
		r.ctx,
		r.stow,
		r.entry.Bucket,
		path,
		r.entry.Threads,
		r.entry.PartSize*Megabyte,
		r.entry.VolumeSize*Megabyte,
	)

	if err != nil {
		return err
	}

	_, err = upload.Send(stream.Out, true)
	if err != nil {
		stream.Out.CloseWithError(err)
		return err
	}

	if err := r.putContents(path+"/contents", included); err != nil {
		return upload.Fail(true, err)
	}

	if err = stream.Wait(); err != nil {
		return err
	}
	return nil
}

func (r *remote) putContents(path string, listing []zed.SnapshotListing) error {
	contents := make([]ArchiveEntry, 0, len(listing))
	for _, v := range listing {
		contents = append(contents, ArchiveEntry{v.Snapshot.Addr.Name, v.Created, v.Identity})
	}

	data, err := json.Marshal(contents)
	if err != nil {
		return err
	}

	if _, err = r.stow.PutObject(r.ctx, r.entry.Bucket, path, data); err != nil {
		return err
	}
	return nil
}

type catalogue map[string]map[int]map[int]string

func (c catalogue) load(listing []string) {
	for _, item := range listing {
		groups := splitPath.FindStringSubmatch(item)
		if len(groups) == 4 {
			if archive, err := strconv.Atoi(groups[2]); err == nil {
				if volume, err := strconv.Atoi(groups[3]); err == nil {
					c.add(groups[1], archive, volume, item)
				}
			}
		}
	}
}

func (c catalogue) add(fs string, archive, volume int, path string) {
	archives, ok := c[fs]
	if !ok {
		archives = make(map[int]map[int]string)
		c[fs] = archives
	}

	if volumes, ok := archives[archive]; !ok {
		archives[archive] = make(map[int]string)
		archives[archive][volume] = path
	} else {
		volumes[volume] = path
	}
}

func (c catalogue) verify(fs string) ([][]string, error) {
	verified := make([][]string, 0)

	unverified, ok := c[fs]
	if !ok {
		return verified, nil
	}

	for i := 0; i < len(unverified); i++ {
		if volumes, ok := unverified[i]; ok {
			verified = append(verified, make([]string, 0))
			for j := 0; j < len(volumes); j++ {
				if path, ok := volumes[j]; ok {
					verified[i] = append(verified[i], path)
				} else {
					return nil, fmt.Errorf("missing volume %d for archive %d for %s", j, i, fs)
				}
			}
		} else {
			return nil, fmt.Errorf("missing archive %d for %s", i, fs)
		}
	}
	return verified, nil
}

// ArchiveEntry represents an item stored in the archive
type ArchiveEntry struct {
	Name     string    `json:"name"`
	Created  time.Time `json:"created"`
	Identity string    `json:"identity"`
}
