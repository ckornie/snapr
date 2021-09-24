package zed

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Addressable provides a mechanism of resolving a file system component address.
type Addressable interface {
	Address() string
}

// Address represents a file system component address (e.g. snapshot).
type Address struct {
	FileSystem FileSystem
	Name       string
}

func (a *Address) asSnapshot() string {
	return a.FileSystem.String() + "@" + a.Name
}

func (a *Address) asBookmark() string {
	return a.FileSystem.String() + "#" + a.Name
}

// NewAddress parses an address from a string using a separator token (e.g. '@' for snapshot).
func NewAddress(address, token string) (*Address, error) {
	splits := strings.SplitN(address, token, 2)
	if len(splits) != 2 {
		return nil, fmt.Errorf("error parsing address '%s'", address)
	}

	name := splits[1]
	fs, err := ToFileSystem(splits[0])
	if err != nil {
		return nil, err
	}

	return &Address{*fs, name}, nil
}

// Destroy will destroy the object at the address.
func (z *Zed) Destroy(ctx context.Context, a Addressable) error {
	cmd := exec.CommandContext(ctx, z.path, "destroy", a.Address())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to destroy '%s': '%s' (%w)", a.Address(), parseError(out), err)
	}
	return nil
}
