package stow

import "regexp"

type commonHeaders struct {
	authorization string
	lastModified  string
	algorithm     string
	securityToken string
	date          string
	credential    string
	signedHeaders string
	signature     string
	contentHash   string
}

var header = commonHeaders{
	authorization: "Authorization",
	lastModified:  "Last-Modified",
	algorithm:     "X-Amz-Algorithm",
	securityToken: "X-Amz-Security-Token",
	date:          "X-Amz-Date",
	credential:    "X-Amz-Credential",
	signedHeaders: "X-Amz-SignedHeaders",
	signature:     "X-Amz-Signature",
	contentHash:   "X-Amz-Content-Sha256",
}

var contentRange = regexp.MustCompile(`bytes\s(?P<begin>\d+)-(?P<end>\d+)\/(?P<size>\d+)`)

const (
	// EmptyHash is the hex encoded SHA256 value of an empty string.
	EmptyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// UnsignedPayload indicates that the request payload body is unsigned.
	UnsignedPayload = "UNSIGNED-PAYLOAD"

	// TimeFormat is the time format to be used in the X-Amz-Date header or query parameter.
	TimeFormat = "20060102T150405Z"

	// DateFormat is the shorten time format used in the credential scope.
	DateFormat = "20060102"

	// SignAlgorithm represents the default hash algorithm.
	SignAlgorithm = "AWS4-HMAC-SHA256"
)
