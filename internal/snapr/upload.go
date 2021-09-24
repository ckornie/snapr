package snapr

import (
	"context"
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"snapr/internal/stow"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

type upload struct {
	ctx        context.Context
	stow       *stow.Stow
	bucket     string
	path       string
	volumeSize int
	volumes    []volume
	free       chan *request
	pending    chan *request
	progress   progress
}

type request struct {
	bucket     string
	key        string
	identifier string
	volume     int
	part       int
	buffer     []byte
	response   *stow.Part
}

func newUpload(ctx context.Context, stow *stow.Stow, bucket, path string, threads, partSize, volumeSize int) (*upload, error) {
	volumes, err := makeVolumes(ctx, stow, bucket, path)
	if err != nil {
		return nil, err
	}

	return &upload{
		ctx:        ctx,
		stow:       stow,
		bucket:     bucket,
		path:       path,
		volumeSize: volumeSize,
		volumes:    volumes,
		free:       makeRequests(threads, partSize),
		pending:    make(chan *request, threads),
		progress:   progress{time.Now(), 0, 0, sha1.New()},
	}, nil
}

func makeRequests(threads, partSize int) chan *request {
	unused := make(chan *request, threads)
	for i := 0; i < threads; i++ {
		req := &request{
			part:   i,
			buffer: make([]byte, 0, partSize),
		}
		unused <- req
	}
	return unused
}

func makeVolumes(ctx context.Context, stow *stow.Stow, bucket, path string) ([]volume, error) {
	v, err := newVolume(ctx, stow, 0, bucket, path)
	if err != nil {
		return nil, err
	}

	volumes := []volume{*v}
	return volumes, nil
}

// Send will upload the source. Interim files will be removed if there's an error and abort is true.
func (u *upload) Send(src io.Reader, abort bool) (*SendDetails, error) {
	eg, ctx := errgroup.WithContext(u.ctx)

	for i := 0; i < cap(u.free); i++ {
		eg.Go(func() error {
			err := u.takeRequests(ctx)
			if err != nil {
				Logger.Error().Err(err).Stack().Msgf("upload failed for %s", u.path)
			}
			return err
		})
	}

	u.progress.start = time.Now()

	for {
		err := u.read(ctx, src)
		if err != nil {
			if err == io.EOF {
				break
			} else if err != io.ErrUnexpectedEOF {
				close(u.pending)
				return nil, u.Fail(abort, err)
			}
		}
	}

	close(u.pending)

	if err := eg.Wait(); err != nil {
		return nil, u.Fail(abort, err)
	}

	close(u.free)

	for part := range u.free {
		u.readResponse(part)
	}

	for _, v := range u.volumes {
		err := u.complete(&v)
		if err != nil {
			return nil, err
		}
	}

	return u.result(), nil
}

func (u *upload) Fail(abort bool, cause error) error {
	if abort {
		if err := u.abort(); err != nil {
			return fmt.Errorf("abort failed: %s following %w", err, cause)
		}
	}
	return cause
}

func (u *upload) abort() error {
	ctx := context.Background()
	for _, v := range u.volumes {
		if !v.aborted {
			if _, err := u.stow.AbortMultipartUpload(ctx, v.bucket, v.key, v.identifier); err != nil {
				return err
			}
			v.aborted = true
		}
	}
	return nil
}

func (u *upload) result() *SendDetails {
	return &SendDetails{
		Bucket:   u.bucket,
		Path:     u.path,
		Parts:    u.progress.parts,
		Bytes:    u.progress.bytes,
		Hash:     u.progress.hash.Sum(nil),
		Duration: time.Now().Sub(u.progress.start),
	}
}

func (u *upload) read(ctx context.Context, src io.Reader) error {
	select {
	case req := <-u.free:
		u.readResponse(req)

		buf := req.buffer

		vol, cap, err := u.volume(cap(buf))
		if err != nil {
			return err
		}

		n, err := io.ReadFull(src, buf[:cap])

		if n > 0 {
			err := u.enqueue(req, vol, n)
			if err != nil {
				return err
			}
		}

		if err != nil {
			return err
		}
	case <-ctx.Done():
		return fmt.Errorf("upload cancelled")
	}
	return nil
}

func (u *upload) volume(max int) (*volume, int, error) {
	last := len(u.volumes) - 1
	cap := min(max, (u.volumeSize - u.volumes[last].progress.bytes))

	if cap == 0 {
		vol, err := newVolume(u.ctx, u.stow, (last + 1), u.bucket, u.path)
		if err != nil {
			return nil, 0, err
		}
		u.volumes = append(u.volumes, *vol)
		cap = min(max, u.volumeSize)
	}
	return &u.volumes[len(u.volumes)-1], cap, nil
}

func (u *upload) enqueue(req *request, vol *volume, read int) error {
	req.buffer = req.buffer[:read]
	req.response = nil

	vol.progress.parts = vol.progress.parts + 1
	vol.progress.bytes = vol.progress.bytes + read
	vol.progress.hash.Write(req.buffer)

	u.progress.parts = u.progress.parts + 1
	u.progress.bytes = u.progress.bytes + read
	u.progress.hash.Write(req.buffer)

	req.bucket = vol.bucket
	req.key = vol.key
	req.identifier = vol.identifier
	req.volume = vol.sequence
	req.part = vol.progress.parts

	u.pending <- req
	return nil
}

func (u *upload) takeRequests(ctx context.Context) error {
	for req := range u.pending {
		res, err := u.stow.UploadPart(
			u.ctx,
			req.bucket,
			req.key,
			req.identifier,
			req.part,
			req.buffer,
		)
		if err != nil {
			return err
		}

		Logger.Debug().Msgf("part %d of volume %d uploaded", req.part, req.volume)

		part := stow.NewPart(req.part, res.Tag)
		req.response = &part

		u.free <- req
	}
	return nil
}

func (u *upload) readResponse(req *request) {
	if req.response != nil {
		u.volumes[req.volume].parts = append(u.volumes[req.volume].parts, *req.response)
	}
}

func (u *upload) complete(vol *volume) error {
	res, err := u.stow.CompleteMultipartUpload(
		u.ctx,
		vol.bucket,
		vol.key,
		vol.identifier,
		vol.parts,
	)
	if err != nil {
		return err
	}
	vol.tag = res.Tag
	return nil
}

type progress struct {
	start time.Time
	parts int
	bytes int
	hash  hash.Hash
}

type volume struct {
	sequence   int
	path       string
	bucket     string
	key        string
	identifier string
	tag        string
	progress   progress
	parts      []stow.Part
	aborted    bool
}

func newVolume(ctx context.Context, stow *stow.Stow, sequence int, bucket, path string) (*volume, error) {
	key := fmt.Sprintf("%s/%s", path, padNumber(sequence))
	res, err := stow.CreateMultipartUpload(ctx, bucket, key)
	if err != nil {
		return nil, err
	}

	return &volume{
		sequence:   sequence,
		path:       path,
		bucket:     res.Bucket,
		key:        res.Key,
		identifier: res.Identifier,
		progress:   progress{time.Now(), 0, 0, sha1.New()},
		parts:      makeParts(),
	}, nil
}

func makeParts() []stow.Part {
	return make([]stow.Part, 0)
}

// SendDetails is used to present send details.
type SendDetails struct {
	Bucket   string
	Path     string
	Parts    int
	Bytes    int
	Hash     []byte
	Duration time.Duration
}

func (r SendDetails) String() string {
	sent := float64(r.Bytes) / Megabyte
	rate := sent / r.Duration.Seconds()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Bucket: %s\n", r.Bucket))
	sb.WriteString(fmt.Sprintf("Path: %s\n", r.Path))
	sb.WriteString(fmt.Sprintf("Parts: %d\n", r.Parts))
	sb.WriteString(fmt.Sprintf("Hash: %x\n", r.Hash))
	sb.WriteString(fmt.Sprintf("Sent: %.2f\n", sent))
	sb.WriteString(fmt.Sprintf("Rate: %.2f MB/s\n", rate))
	return sb.String()
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}
