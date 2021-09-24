package zed

import (
	"fmt"
	"os/exec"
	"regexp"
)

const creationTime = "Mon Jan _2 15:04 2006"

var sanitizer = regexp.MustCompile(`\r?\n`)

// Zed exposes ZFS operations by wrapping the command-line 'zfs' utility.
type Zed struct {
	path string
}

// New instantiates Zed.
func New() (*Zed, error) {
	path, err := exec.LookPath("zfs")
	if err != nil {
		return nil, fmt.Errorf("could not locate executable (%w)", err)
	}
	return &Zed{
		path: path,
	}, nil
}

func parseError(msg []byte) string {
	return sanitizer.ReplaceAllString(string(msg), " ")
}
