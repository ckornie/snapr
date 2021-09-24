package zed

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// Receive performs a receive.
func (z *Zed) Receive(ctx context.Context, target string, src io.Reader) error {
	cmd := exec.CommandContext(ctx, z.path, "receive", "-d", target)
	cmd.Stdin = src
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not receive stream to '%s': %s (%w)", target, parseError(out), err)
	}
	return nil
}
