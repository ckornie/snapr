package snapr

import (
	"fmt"
	"os"

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
