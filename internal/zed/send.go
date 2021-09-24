package zed

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Stream exposes a reader containing the stream and allows waiting for completion.
type Stream struct {
	cmd        *exec.Cmd
	in         *io.PipeWriter
	Out        *io.PipeReader
	completion func(err error) error
	error      error
	done       bool
	eg         *errgroup.Group
	mu         sync.Mutex
}

func (z *Zed) newStream(ctx context.Context, source *Snapshot, target Snapshot, completion func(error) error) *Stream {
	out, in := io.Pipe()

	eg, ctx := errgroup.WithContext(ctx)

	return &Stream{
		cmd:        z.sendCmd(ctx, source, target),
		in:         in,
		Out:        out,
		completion: completion,
		eg:         eg,
	}
}

func (z *Zed) sendCmd(ctx context.Context, source *Snapshot, target Snapshot) *exec.Cmd {
	if source == nil {
		return exec.CommandContext(ctx, z.path, "send", "--raw", "--holds", "--replicate", target.Address())
	}
	return exec.CommandContext(ctx, z.path, "send", "--raw", "--holds", "--replicate", "-I", source.Address(), target.Address())
}

// Wait allows a client to wait for completion.
func (w *Stream) Wait() error {
	return w.eg.Wait()
}

func (w *Stream) run() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.done {
		return errors.New("command failed: has been run previously")
	}

	w.done = true

	var stderr bytes.Buffer
	w.cmd.Stderr = &stderr

	stdout, err := w.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = w.cmd.Start()
	if err != nil {
		return err
	}

	w.eg.Go(
		func() error {
			defer w.in.Close()

			var cause error

			_, err := io.Copy(w.in, stdout)
			if err != nil {
				cause = fmt.Errorf("pipe failed: %w", err)
				w.in.CloseWithError(cause)

				if err := w.cmd.Process.Kill(); err != nil {
					cause = fmt.Errorf("command termination failed: %s (%w)", err, cause)
				}
			} else {
				err = w.cmd.Wait()
				if err != nil {
					cause = fmt.Errorf("command failed: %w", err)
					w.in.CloseWithError(cause)
				}
			}

			if err := w.completion(cause); err != nil {
				cause = fmt.Errorf("completion failed: %s (%w)", err, cause)
			}

			w.error = cause
			return cause
		},
	)
	return nil
}

// Send returns a stream.
func (z *Zed) Send(ctx context.Context, source *Snapshot, target Snapshot, completion func(error) error) (*Stream, error) {
	stream := z.newStream(ctx, source, target, completion)
	return stream, stream.run()
}
