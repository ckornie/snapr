package stow

import (
	"net/http"
	"testing"
	"time"
)

func TestAuth(t *testing.T) {
	url := "https://my-precious-bucket.s3.amazonaws.com"
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Errorf("could not form request: %v", err)
	}

	authenticator := authenticator{
		service: "s3",
		region:  "us-east-1",
		key:     "AKIAIOSFODNN7EXAMPLE",
		secret:  "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		time: func() time.Time {
			return time.Date(2015, time.September, 15, 12, 45, 0, 0, time.UTC)
		},
	}

	authenticator.noBody(req)

	expected := "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20150915/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=182072eb53d85c36b2d791a1fa46a12d23454ec1e921b02075c23aee40166d5a"
	actual := req.Header.Get(header.authorization)

	if actual != expected {
		t.Errorf("failed with %s", actual)
	}
}
