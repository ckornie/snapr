package stow

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type clock func() time.Time

func systemClock() clock {
	return func() time.Time {
		return time.Now().UTC()
	}
}

type authHeaders struct {
	signed    string
	canonical string
	hash      string
}

type authRequest struct {
	method   string
	uri      string
	query    string
	headers  authHeaders
	bodyHash string
}

type authScope struct {
	date    string
	region  string
	service string
}

type authorization struct {
	algorithm string
	time      string
	scope     authScope
	request   authRequest
}

type authenticator struct {
	service string
	region  string
	key     string
	secret  string
	time    clock
}

func newAuthenticator(region, key, secret string) authenticator {
	return authenticator{
		service: "s3",
		region:  region,
		key:     key,
		secret:  secret,
		time:    systemClock(),
	}
}

func (a authenticator) noBody(req *http.Request) {
	req.Header.Add(header.contentHash, EmptyHash)
	now := a.time()
	req.Header.Add(header.date, now.Format(TimeFormat))
	req.Header.Add(header.authorization, a.newAuthorization(req, now, EmptyHash).authorize(a.key, a.secret))
}

func (a authenticator) withBody(req *http.Request, b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("no body")
	}

	now := a.time()
	req.Header.Add(header.date, now.Format(TimeFormat))

	hash := fmt.Sprintf("%x", sha256.Sum256(b))
	req.Header.Add(header.contentHash, hash)

	req.Header.Add(header.authorization, a.newAuthorization(req, now, hash).authorize(a.key, a.secret))
	return nil
}

func (a authenticator) newAuthorization(req *http.Request, now time.Time, hash string) authorization {
	return newAuthorization(now, a.region, newAuthRequest(req, hash))
}

func (a authRequest) text() string {
	return strings.Join([]string{
		a.method,
		a.uri,
		a.query,
		a.headers.canonical,
		a.headers.signed,
		a.bodyHash,
	}, "\n")
}

func (a authRequest) hash() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(a.text())))
}

func (s authScope) text() string {
	return strings.Join([]string{
		s.date,
		s.region,
		s.service,
		"aws4_request",
	}, "/")
}

func (a authorization) sign(secret string) string {
	text := strings.Join([]string{
		a.algorithm,
		a.time,
		a.scope.text(),
		a.request.hash(),
	}, "\n")

	hmac := []byte("AWS4" + secret)
	hmac = hash(hmac, a.scope.date)
	hmac = hash(hmac, a.scope.region)
	hmac = hash(hmac, a.scope.service)
	hmac = hash(hmac, "aws4_request")
	return hex.EncodeToString(hash(hmac, text))
}

func newAuthRequest(req *http.Request, hash string) authRequest {
	return authRequest{
		method:   req.Method,
		uri:      uri(req.URL),
		query:    query(req.URL),
		headers:  newAuthHeaders(req),
		bodyHash: hash,
	}
}

func newAuthHeaders(req *http.Request) (headers authHeaders) {
	copy := make(http.Header)

	copy["host"] = append(copy["host"], host(req))

	if req.ContentLength > 0 {
		copy["content-length"] = append(copy["content-length"], strconv.FormatInt(req.ContentLength, 10))
	}

	hash := EmptyHash
	for k, v := range req.Header {
		if k == header.contentHash {
			hash = v[0]
		}

		key := strings.ToLower(k)

		if _, ok := copy[key]; ok {
			copy[key] = append(copy[key], v...)
		} else {
			copy[key] = v
		}
	}

	keys := make([]string, 0, len(copy))
	for k := range copy {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, v := range keys {
		sb.WriteString(v)
		sb.WriteRune(':')
		sb.WriteString(strings.Join(copy[v], ","))
		sb.WriteRune('\n')
	}
	return authHeaders{strings.Join(keys, ";"), sb.String(), hash}
}

func newAuthorization(now time.Time, region string, request authRequest) authorization {
	return authorization{
		SignAlgorithm,
		now.Format(TimeFormat),
		newAuthScope(now, region),
		request,
	}
}

func newAuthScope(now time.Time, region string) authScope {
	return authScope{
		now.Format(DateFormat),
		region,
		"s3",
	}
}

func host(req *http.Request) string {
	host := req.URL.Host
	if len(req.Host) > 0 {
		host = req.Host
	}
	return host
}

func uri(u *url.URL) string {
	uri := u.EscapedPath()
	if len(uri) == 0 {
		uri = "/"
	}
	return uri
}

func query(u *url.URL) string {
	query := u.Query()
	for key := range query {
		sort.Strings(query[key])
	}

	var raw strings.Builder
	raw.WriteString(strings.Replace(query.Encode(), "+", "%20", -1))
	return raw.String()
}

func (a authorization) authorize(key, secret string) string {
	var sb strings.Builder
	sb.WriteString(SignAlgorithm)
	sb.WriteRune(' ')
	sb.WriteString("Credential=")
	sb.WriteString(key)
	sb.WriteRune('/')
	sb.WriteString(a.scope.text())
	sb.WriteString(", ")
	sb.WriteString("SignedHeaders=")
	sb.WriteString(a.request.headers.signed)
	sb.WriteString(", ")
	sb.WriteString("Signature=")
	sb.WriteString(a.sign(secret))
	return sb.String()
}

func hash(key []byte, data string) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write([]byte(data))
	return hash.Sum(nil)
}
