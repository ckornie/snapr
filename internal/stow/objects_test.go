package stow

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentRange(t *testing.T) {
	header := http.Header{}
	header.Add("Content-Range", "bytes 0-9/443")
	begin, end, size, err := parseContentRange(header)
	assert.Nil(t, err)
	assert.Equal(t, begin, 0)
	assert.Equal(t, end, 9)
	assert.Equal(t, size, 443)
}
