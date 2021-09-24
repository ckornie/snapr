package snapr

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

const (
	// Megabyte is the number of bytes in a megabyte.
	Megabyte = 1_000_000

	// Threads is the default number of concurrent uploads.
	Threads = 10

	// PartSize is the default part size in megabytes.
	PartSize = 100_000

	// VolumeSize is the default volume size in megabytes.
	VolumeSize = 200
)

// Logger is the default logger for the package.
var Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

func findVolumes(fs string, listing []string) (map[int][]string, error) {
	prefix := fs + "/"
	volumes := make(map[int][]string, 0)

	for _, item := range listing {
		if strings.HasPrefix(item, fs) {
			split := strings.SplitN(strings.TrimPrefix(item, prefix), "/", 2)
			if len(split) > 0 {
				seq, err := strconv.Atoi(split[0])
				if err == nil {
					v, ok := volumes[seq]
					if !ok {
						v = make([]string, 0)
					}
					volumes[seq] = append(v, item)
				}
			}
		}
	}

	if len(volumes) == 0 {
		return nil, fmt.Errorf("no volumes found for %s", fs)
	}

	return volumes, nil
}

func padNumber(number int) string {
	return fmt.Sprintf("%05d", number)
}

func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
