package stow

import (
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// CreateBucketRequest is used to model the namesake request.
type CreateBucketRequest struct {
	ctx                context.Context
	Bucket             string
	XMLName            xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CreateBucketConfiguration"`
	LocationConstraint string   `xml:"LocationConstraint,omitempty"`
}

func (r CreateBucketRequest) formRequest(factory requestFactory, p Provider) (*http.Request, error) {
	m, err := xml.Marshal(r)
	if err != nil {
		return nil, err
	}
	return factory(r.ctx, http.MethodPut, p.urlBucket(r.Bucket), m)
}

// CreateBucketResponse used to model the namesake response.
type CreateBucketResponse struct {
	Location string
	Metadata Metadata
}

// DeleteBucketRequest is used to model the namesake request.
type DeleteBucketRequest struct {
	ctx    context.Context
	Bucket string
}

func (r DeleteBucketRequest) formRequest(factory requestFactory, p Provider) (*http.Request, error) {
	return factory(r.ctx, http.MethodDelete, p.urlBucket(r.Bucket), nil)
}

// DeleteBucketResponse used to model the namesake response.
type DeleteBucketResponse struct {
	Metadata Metadata
}

// ListBucketsResponse used to model the namesake response.
type ListBucketsResponse struct {
	XMLName   xml.Name `xml:"ListAllMyBucketsResult"`
	Owner     string   `xml:"Owner>ID"`
	OwnerName string   `xml:"Owner>DisplayName"`
	Buckets   []Bucket `xml:"Buckets>Bucket"`
	Metadata  Metadata
}

// Bucket represents a bucket.
type Bucket struct {
	Name         string    `xml:"Name"`
	CreationDate time.Time `xml:"CreationDate"`
}

// CreateBucket will create a bucket (see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucket.html).
func (s *Stow) CreateBucket(name string) (*CreateBucketResponse, error) {
	res, err := s.doOperation(
		CreateBucketRequest{
			Bucket:             name,
			LocationConstraint: s.provider.Region,
		},
	)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, newStatusError(fmt.Sprintf("failed creating bucket '%s'", name), *res, b)
	}

	return &CreateBucketResponse{
		Location: res.Header.Get("Location"),
		Metadata: newMetadata(res),
	}, nil
}

// DeleteBucket will delete a bucket (see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucket.html).
func (s *Stow) DeleteBucket(ctx context.Context, name string) (*DeleteBucketResponse, error) {
	res, err := s.doOperation(DeleteBucketRequest{ctx, name})

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 204 {
		return nil, newStatusError(fmt.Sprintf("failed deleting bucket '%s'", name), *res, b)
	}

	return &DeleteBucketResponse{
		Metadata: newMetadata(res),
	}, nil
}

// ListBuckets will list all buckets (see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListBuckets.html).
func (s *Stow) ListBuckets(ctx context.Context) (*ListBucketsResponse, error) {
	res, err := s.doOperation(get{ctx})

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, newStatusError("failed listing buckets", *res, b)
	}

	r := &ListBucketsResponse{
		Metadata: newMetadata(res),
	}

	if err := xml.Unmarshal(b, r); err != nil {
		return nil, err
	}
	return r, nil
}
