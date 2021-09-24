package stow

import (
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

// CreateMultipartUploadRequest is used to model the namesake request.
type CreateMultipartUploadRequest struct {
	ctx    context.Context
	Bucket string
	Key    string
}

func (r CreateMultipartUploadRequest) formRequest(factory requestFactory, p Provider) (*http.Request, error) {
	query := p.urlBucket(r.Bucket) + "/" + r.Key + "?uploads"
	return factory(r.ctx, http.MethodPost, query, nil)
}

// CreateMultipartUploadResponse used to model the namesake response.
type CreateMultipartUploadResponse struct {
	XMLName    xml.Name `xml:"InitiateMultipartUploadResult"`
	Bucket     string   `xml:"Bucket"`
	Key        string   `xml:"Key"`
	Identifier string   `xml:"UploadId"`
	Metadata   Metadata
}

// UploadPartRequest is used to model the namesake request.
type UploadPartRequest struct {
	ctx        context.Context
	Bucket     string
	Key        string
	Identifier string
	PartNumber int
	Data       []byte
}

func (r UploadPartRequest) formRequest(factory requestFactory, p Provider) (*http.Request, error) {
	query := p.urlBucket(r.Bucket) + "/" + r.Key + "?partNumber=" + strconv.Itoa(r.PartNumber) + "&uploadId=" + url.QueryEscape(r.Identifier)
	return factory(r.ctx, http.MethodPut, query, r.Data)
}

// UploadPartResponse used to model the namesake response.
type UploadPartResponse struct {
	Tag      string
	Metadata Metadata
}

// UploadPartCopyRequest is used to model the namesake request.
type UploadPartCopyRequest struct {
	ctx        context.Context
	Bucket     string
	Key        string
	Identifier string
	PartNumber int
	Source     string
	From       int
	To         int
}

func (r UploadPartCopyRequest) formRequest(factory requestFactory, p Provider) (*http.Request, error) {
	query := p.urlBucket(r.Bucket) + "/" + r.Key + "?partNumber=" + strconv.Itoa(r.PartNumber) + "&uploadId=" + url.QueryEscape(r.Identifier)
	req, err := factory(r.ctx, http.MethodPut, query, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-Amz-Copy-Source", r.Source)
	source := "bytes=" + strconv.Itoa(r.From) + "-" + strconv.Itoa(r.To)
	req.Header.Add("X-Amz-Copy-Source-Range", source)
	return req, nil
}

// UploadPartCopyResponse used to model the namesake response.
type UploadPartCopyResponse struct {
	XMLName      xml.Name  `xml:"CopyPartResult"`
	Tag          string    `xml:"ETag"`
	LastModified time.Time `xml:"LastModified"`
	Metadata     Metadata
}

// CompleteMultipartUploadRequest is used to model the namesake request.
type CompleteMultipartUploadRequest struct {
	ctx        context.Context
	XMLName    xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CompleteMultipartUpload"`
	Bucket     string
	Key        string
	Identifier string
	Parts      []Part `xml:"Part,omitempty"`
}

func (r CompleteMultipartUploadRequest) formRequest(factory requestFactory, p Provider) (*http.Request, error) {
	m, err := xml.Marshal(r)
	if err != nil {
		return nil, err
	}
	query := p.urlBucket(r.Bucket) + "/" + r.Key + "?uploadId=" + url.QueryEscape(r.Identifier)
	return factory(r.ctx, http.MethodPost, query, m)
}

// CompleteMultipartUploadResponse used to model the namesake response.
type CompleteMultipartUploadResponse struct {
	XMLName  xml.Name `xml:"CompleteMultipartUploadResult"`
	Location string   `xml:"Location"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	Tag      string   `xml:"ETag"`
	Metadata Metadata
}

// AbortMultipartUploadRequest is used to model the namesake request.
type AbortMultipartUploadRequest struct {
	ctx        context.Context
	Bucket     string
	Key        string
	Identifier string
}

func (r AbortMultipartUploadRequest) formRequest(factory requestFactory, p Provider) (*http.Request, error) {
	query := p.urlBucket(r.Bucket) + "/" + r.Key + "?uploadId=" + url.QueryEscape(r.Identifier)
	return factory(r.ctx, http.MethodDelete, query, nil)
}

// AbortMultipartUploadResponse used to model the namesake response.
type AbortMultipartUploadResponse struct {
	Metadata Metadata
}

// Part represents a part used for completion.
type Part struct {
	PartNumber int    `xml:"PartNumber"`
	Tag        string `xml:"ETag"`
}

// NewPart instantiates a Part
func NewPart(partNumber int, tag string) Part {
	return Part{partNumber, tag}
}

// CreateMultipartUpload will create a multi-part upload (see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateMultipartUpload.html).
func (s *Stow) CreateMultipartUpload(ctx context.Context, bucket, key string) (*CreateMultipartUploadResponse, error) {
	res, err := s.doOperation(
		CreateMultipartUploadRequest{
			ctx:    ctx,
			Bucket: bucket,
			Key:    key,
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
		return nil, newStatusError(fmt.Sprintf("failed creating multi-part upload '%s' in bucket '%s'\n", key, bucket), *res, b)
	}

	response := &CreateMultipartUploadResponse{
		Metadata: newMetadata(res),
	}

	if err := xml.Unmarshal(b, response); err != nil {
		return nil, err
	}
	return response, nil
}

// UploadPart will upload a part of a multi-part upload (see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_UploadPart.html).
func (s *Stow) UploadPart(ctx context.Context, bucket, key, upload string, partNumber int, data []byte) (*UploadPartResponse, error) {
	res, err := s.doOperation(
		UploadPartRequest{
			ctx:        ctx,
			Bucket:     bucket,
			Key:        key,
			Identifier: upload,
			PartNumber: partNumber,
			Data:       data,
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
		return nil, newStatusError(fmt.Sprintf("failed uploading part %d of '%s'\n", partNumber, key), *res, b)
	}

	response := &UploadPartResponse{
		Tag:      res.Header.Get("ETag"),
		Metadata: newMetadata(res),
	}
	return response, nil
}

// UploadPartCopy will copy a part from an existing object (see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_UploadPartCopy.html).
func (s *Stow) UploadPartCopy(ctx context.Context, bucket, key, upload string, partNumber int, source string, from, to int) (*UploadPartCopyResponse, error) {
	res, err := s.doOperation(
		UploadPartCopyRequest{
			ctx:        ctx,
			Bucket:     bucket,
			Key:        key,
			Identifier: upload,
			PartNumber: partNumber,
			Source:     source,
			From:       from,
			To:         to,
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
		return nil, newStatusError(fmt.Sprintf("failed copying part %d of '%s'\n", partNumber, key), *res, b)
	}

	response := &UploadPartCopyResponse{
		Metadata: newMetadata(res),
	}

	if err := xml.Unmarshal(b, response); err != nil {
		return nil, err
	}
	return response, nil
}

// CompleteMultipartUpload will complete a multi-part upload (see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_CompleteMultipartUpload.html).
func (s *Stow) CompleteMultipartUpload(ctx context.Context, bucket, key, identifier string, parts []Part) (*CompleteMultipartUploadResponse, error) {
	sort.SliceStable(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber
	})

	res, err := s.doOperation(
		CompleteMultipartUploadRequest{
			ctx:        ctx,
			Bucket:     bucket,
			Key:        key,
			Identifier: identifier,
			Parts:      parts,
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
		return nil, newStatusError(fmt.Sprintf("failed completing multi-part upload '%s' to bucket '%s'", key, bucket), *res, b)
	}

	response := &CompleteMultipartUploadResponse{
		Metadata: newMetadata(res),
	}

	if err := xml.Unmarshal(b, response); err != nil {
		return nil, err
	}
	return response, nil
}

// AbortMultipartUpload will abort a multi-part upload (see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_AbortMultipartUpload.html).
func (s *Stow) AbortMultipartUpload(ctx context.Context, bucket, key, identifier string) (*AbortMultipartUploadResponse, error) {
	res, err := s.doOperation(
		AbortMultipartUploadRequest{
			ctx:        ctx,
			Bucket:     bucket,
			Key:        key,
			Identifier: identifier,
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

	if res.StatusCode != 204 {
		return nil, newStatusError(fmt.Sprintf("failed aborting multi-part upload '%s' in bucket '%s'", key, bucket), *res, b)
	}

	return &AbortMultipartUploadResponse{
		Metadata: newMetadata(res),
	}, nil
}
