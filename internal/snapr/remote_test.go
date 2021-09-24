package snapr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCatalogue(t *testing.T) {
	fs := "pool-0/test"
	listing := []string{
		fs + "/00000/contents",
		fs + "/00000/00000",
		fs + "/00000/contents",
		fs + "/00000/00000",
		fs + "/00000/00001",
		fs + "/00000/00002",
		fs + "/00001/contents",
		fs + "/00001/00000",
		fs + "/00001/00001",
	}

	catalogue := make(catalogue)
	catalogue.load(listing)
	verified, err := catalogue.verify(fs)

	assert.NoError(t, err)
	assert.Len(t, verified, 2)
	assert.Contains(t, verified[0], fs+"/00000/00000", fs+"/00000/00001", fs+"/00000/00002")
	assert.Contains(t, verified[1], fs+"/00001/00000", fs+"/00001/00001")
}

func TestCatalogueWithMissingArchive(t *testing.T) {
	fs := "pool-0/test"
	listing := []string{
		fs + "/00000/contents",
		fs + "/00000/00000",
		fs + "/00000/00001",
		fs + "/00000/00002",
		fs + "/00002/contents",
		fs + "/00002/00000",
		fs + "/00002/00001",
	}

	catalogue := make(catalogue)
	catalogue.load(listing)
	_, err := catalogue.verify(fs)

	assert.Error(t, err)
}

func TestCatalogueWithMissingVolume(t *testing.T) {
	fs := "pool-0/test"
	listing := []string{
		fs + "/00000/contents",
		fs + "/00000/00000",
		fs + "/00000/00002",
		fs + "/00000/00003",
		fs + "/00002/contents",
		fs + "/00002/00000",
		fs + "/00002/00001",
	}

	catalogue := make(catalogue)
	catalogue.load(listing)
	_, err := catalogue.verify(fs)

	assert.Error(t, err)
}
