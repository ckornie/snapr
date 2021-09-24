package snapr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindVolumes(t *testing.T) {
	listing := []string{
		"pool-0/example/00000/00000",
		"pool-0/test/00000/00000",
		"pool-0/test/00000/00001",
		"pool-0/test/00001/00000",
		"pool-0/test/00001/00001",
	}

	volumes, err := findVolumes("pool-0/test", listing)
	assert.NoError(t, err)
	assert.Len(t, volumes, 2)

	assert.Contains(t, volumes, 0)
	assert.Len(t, volumes[0], 2)
	assert.Equal(t, volumes[0][0], "pool-0/test/00000/00000")
	assert.Contains(t, volumes[0][1], "pool-0/test/00000/00001")

	assert.Contains(t, volumes, 1)
	assert.Len(t, volumes[1], 2)
	assert.Equal(t, volumes[1][0], "pool-0/test/00001/00000")
	assert.Contains(t, volumes[1][1], "pool-0/test/00001/00001")

	_, err = findVolumes("pool-1/test", listing)
	assert.Error(t, err)
}
