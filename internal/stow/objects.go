package stow

import (
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PutObjectRequest is used to model the namesake request.
type PutObjectRequest struct {
	ctx    context.Context
	Bucket string
	Key    string
	Data   []byte
}

func (r PutObjectRequest) formRequest(factory requestFactory, p Provider) (*http.Request, error) {
	query := p.urlBucket(r.Bucket) + "/" + r.Key
	return factory(r.ctx, http.MethodPut, query, r.Data)
}

// PutObjectResponse used to model the namesake response.
type PutObjectResponse struct {
	Tag      string
	Metadata Metadata
}

// ListObjectsRequest is used to model the namesake request.
type ListObjectsRequest struct {
	ctx          context.Context
	Bucket       string
	Continuation string
}

func (r ListObjectsRequest) formRequest(factory requestFactory, p Provider) (*http.Request, error) {
	query := p.urlBucket(r.Bucket) + "/?list-type=2"
	if r.Continuation != "" {
		query = query + "&continuation-token=" + url.QueryEscape(r.Continuation)
	}
	return factory(r.ctx, http.MethodGet, query, nil)
}

// ListObjectsResponse used to model the namesake response.
type ListObjectsResponse struct {
	XMLName      xml.Name `xml:"ListBucketResult"`
	Name         string   `xml:"Name"`
	KeyCount     int      `xml:"KeyCount"`
	MaxKeys      int      `xml:"MaxKeys"`
	Truncated    bool     `xml:"IsTruncated"`
	Continuation string   `xml:"NextContinuationToken"`
	Objects      []Object `xml:"Contents"`
	Metadata     Metadata
}

// GetObjectRequest is used to model the namesake request.
type GetObjectRequest struct {
	ctx    context.Context
	Bucket string
	Path   string
	Begin  int
	End    int
}

func (r GetObjectRequest) formRequest(factory requestFactory, p Provider) (*http.Request, error) {
	query := p.urlBucket(r.Bucket) + "/" + r.Path
	req, err := factory(r.ctx, http.MethodGet, query, nil)
	if err != nil {
		return nil, err
	}

	if r.End > r.Begin {
		req.Header.Add("Range", "bytes="+strconv.Itoa(r.Begin)+"-"+strconv.Itoa(r.End))
	}
	return req, nil
}

// GetObjectResponse used to model the namesake response.
type GetObjectResponse struct {
	Tag      string
	Modified time.Time
	Begin    int
	End      int
	Size     int
	Content  []byte
	Metadata Metadata
}

// Object records a stored object's properties.
type Object struct {
	Key          string    `xml:"Key"`
	CreationDate time.Time `xml:"LastModified"`
	Tag          string    `xml:"ETag"`
	Size         int       `xml:"Size"`
	StorageClass string    `xml:"StorageClass"`
}

// PutObject will upload an object to a bucket (see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObject.html).
func (s *Stow) PutObject(ctx context.Context, bucket, key string, data []byte) (*PutObjectResponse, error) {
	res, err := s.doOperation(PutObjectRequest{ctx, bucket, key, data})

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, newStatusError("failed listing objects", *res, b)
	}

	response := &PutObjectResponse{
		Tag:      res.Header.Get("ETag"),
		Metadata: newMetadata(res),
	}
	return response, nil
}

// ListObjects will list objects in a bucket (see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html).
func (s *Stow) ListObjects(ctx context.Context, bucket, continuation string) (*ListObjectsResponse, error) {
	res, err := s.doOperation(ListObjectsRequest{ctx, bucket, continuation})

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, newStatusError("failed listing objects", *res, b)
	}

	response := &ListObjectsResponse{
		Metadata: newMetadata(res),
	}

	if err := xml.Unmarshal(b, response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetObject will get an object (see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObject.html).
func (s *Stow) GetObject(ctx context.Context, bucket, path string, begin, end int) (*GetObjectResponse, error) {
	res, err := s.doOperation(GetObjectRequest{ctx, bucket, path, begin, end})

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 && res.StatusCode != 206 {
		return nil, newStatusError(fmt.Sprintf("failed getting object '%s' from '%s'", path, bucket), *res, b)
	}

	response := &GetObjectResponse{
		Tag:      res.Header.Get("ETag"),
		Metadata: newMetadata(res),
	}

	modified, err := time.Parse(time.RFC1123, res.Header.Get(header.lastModified))
	if err == nil {
		response.Modified = modified
	}

	response.Begin, response.End, response.Size, _ = parseContentRange(res.Header)
	response.Content = b
	return response, nil
}

// ListAllObjects is a helper method which assembles a full listing using pagination.
func (s *Stow) ListAllObjects(ctx context.Context, bucket string) (*ListObjectsResponse, error) {
	response := &ListObjectsResponse{
		Metadata: Metadata{},
		Objects:  make([]Object, 0),
	}

	continuation := ""
	for {
		res, err := s.ListObjects(ctx, bucket, continuation)
		if err != nil {
			return nil, err
		}

		response.Name = res.Name
		response.KeyCount = response.KeyCount + res.KeyCount
		response.Objects = append(response.Objects, res.Objects...)
		continuation = res.Continuation

		if !res.Truncated {
			break
		}
	}
	return response, nil
}

// ListAllKeys is a helper method which assembles a listing of keys using pagination.
func (s *Stow) ListAllKeys(ctx context.Context, bucket string) ([]string, error) {
	res, err := s.ListAllObjects(ctx, bucket)
	if err != nil {
		return nil, err
	}

	listing := make([]string, 0)
	for _, object := range res.Objects {
		listing = append(listing, object.Key)
	}
	return listing, nil
}

func parseContentRange(headers http.Header) (begin, end, size int, err error) {
	match := contentRange.FindStringSubmatch(headers.Get("Content-Range"))
	if len(match) == 4 {
		if begin, err = strconv.Atoi(match[1]); err != nil {
			return 0, 0, 0, fmt.Errorf("could not parse content range: %w", err)
		}
		if end, err = strconv.Atoi(match[2]); err != nil {
			return 0, 0, 0, fmt.Errorf("could not parse content range: %w", err)
		}
		if size, err = strconv.Atoi(match[3]); err != nil {
			return 0, 0, 0, fmt.Errorf("could not parse content range: %w", err)
		}
		return begin, end, size, nil
	}
	return 0, 0, 0, fmt.Errorf("could not parse content range")
}
