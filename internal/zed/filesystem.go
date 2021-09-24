package zed

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// FileSystem represents a filesystem
type FileSystem struct {
	Pool string
	Name string
}

func (f FileSystem) String() string {
	return f.Pool + "/" + f.Name
}

// ToFileSystem parses a string address into a file system.
func ToFileSystem(fs string) (*FileSystem, error) {
	splits := strings.SplitN(fs, "/", 2)
	if len(splits) != 2 {
		return nil, fmt.Errorf("error parsing '%s'", fs)
	}
	return &FileSystem{splits[0], splits[1]}, nil
}

// SetProperty will set a user property on the file system.
func (z *Zed) SetProperty(ctx context.Context, fs FileSystem, domain, property, value string) error {
	cmd := exec.CommandContext(ctx, z.path, "set", domain+":"+property+"="+value, fs.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set property '%s:%s': %s (%w)", domain, property, parseError(out), err)
	}
	return nil
}
