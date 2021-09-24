package snapr

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReadSettings(t *testing.T) {

	raw := `
	{
		"fileSystems": {
		  "pool-0/test": {
				"snap":[
					{
						"interval": "23h30m",
						"prefix": "daily",
						"hold": ["backblaze"]
					},
					{
						"interval": "30m",
						"prefix": "hourly"
					}
				],
				"send":[
					{
					"endpoint": "s3.eu-central-003.backblazeb2.com",
					"region": "eu-central-003",
					"account": "123456789",
					"secret": "SSSSHH",
					"bucket": "bucket",
					"release": ["backblaze"]
					}
				]
		  	}
		},
		"threads": 20,
		"volumeSize": 150000,
		"partSize": 200
	}
	`

	settings := Settings{
		Threads:    10,
		VolumeSize: 100_000,
		PartSize:   100,
	}

	err := json.Unmarshal([]byte(raw), &settings)
	assert.NoError(t, err)

	assert.Equal(t, 20, settings.Threads)
	assert.Equal(t, 150_000, settings.VolumeSize)
	assert.Equal(t, 200, settings.PartSize)
	assert.Contains(t, settings.FileSystems, "pool-0/test")

	target := settings.FileSystems["pool-0/test"]

	assert.Len(t, target.Snap, 2)

	assert.Equal(t, "daily", target.Snap[0].Prefix)
	interval, err := target.Snap[0].IntervalDuration()
	assert.NoError(t, err)
	assert.Equal(t, (23*time.Hour)+(30*time.Minute), interval)
	assert.Len(t, target.Snap[0].Hold, 1)
	assert.Contains(t, target.Snap[0].Hold, "backblaze")

	assert.Equal(t, "hourly", target.Snap[1].Prefix)
	interval, err = target.Snap[1].IntervalDuration()
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Minute, interval)

	assert.Len(t, target.Send, 1)
	assert.Equal(t, "s3.eu-central-003.backblazeb2.com", target.Send[0].Endpoint)
	assert.Equal(t, "eu-central-003", target.Send[0].Region)
	assert.Equal(t, "123456789", target.Send[0].Account)
	assert.Equal(t, "SSSSHH", target.Send[0].Secret)
	assert.Equal(t, "bucket", target.Send[0].Bucket)
	assert.Equal(t, "backblaze", target.Send[0].Release[0])
}
