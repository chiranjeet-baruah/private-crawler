package paapi5

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Semantics3/go-crawler/sources/amazon/paapi5/types"
)

type (
	// Request
	request struct {
		headers     *headers
		payload     []byte
		operation   types.Operation
		client      *Client
		httpRequest *http.Request
		dateTime    time.Time
		amzDate     string
		amzDateTime string
		error       error
		ctx         context.Context
	}
	// Headers represents header data needed to be sent
	headers struct {
		keys             []string
		values           map[string]string
		canonicalHeaders string
		signedHeaders    string
		credentialScope  string
	}
)

// constants for mandatory header and required for generating signature
const (
	headerAccept          string = "application/json, text/javascript"
	headerContentType     string = "application/json; charset=UTF-8"
	headerContentEncoding string = "amz-1.0"
	authAlgorithm         string = "AWS4-HMAC-SHA256"
	amzDateTimeFormat     string = "20060102T150405Z"
	amzDateFormat         string = "20060102"
	serviceName           string = "ProductAdvertisingAPI"
	aws4Request           string = "aws4_request" // Termination String
)

// getEndpoint - construct API endpoint
func (r *request) getEndpoint() string {
	scheme := r.client.locale.Scheme()
	host := r.client.locale.Host()
	path := r.operation.GetPath()
	endpoint := fmt.Sprintf("%s://%s%s", scheme, host, path)
	return endpoint
}

func (r *request) build() *request {
	r.amzDate = r.dateTime.Format(amzDateFormat)
	r.amzDateTime = r.dateTime.Format(amzDateTimeFormat)
	endpoint := r.getEndpoint()

	// Construct HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(r.payload))
	if err != nil {
		r.error = err
		return r
	}

	if r.ctx != nil {
		req = req.WithContext(r.ctx)
	}

	// set request and headers
	r.httpRequest = req
	r.setHeaders()

	return r
}

// setHeaders sets header values in request and generates canonical headers and signed header string
// reference: https://webservices.amazon.com/paapi5/documentation/sending-request.html#headers
func (r *request) setHeaders() {
	headerValues := map[string]string{
		"content-encoding": headerContentEncoding,
		"content-type":     headerContentType,
		"host":             r.client.locale.Host(),
		"x-amz-date":       r.amzDateTime,
		"x-amz-target":     r.operation.GetTarget(),
	}
	// get header keys from above map
	headerKeys := make([]string, len(headerValues))
	i := 0
	for k := range headerValues {
		headerKeys[i] = k
		i++
	}
	// sort keys
	sort.Strings(headerKeys)

	// set request headers
	for key, val := range headerValues {
		r.httpRequest.Header.Set(key, val)
	}

	// get canonical headers for signing
	headerKeyVals := make([]string, len(headerKeys))
	for idx, key := range headerKeys {
		headerKeyVals[idx] = fmt.Sprintf("%s:%s", key, headerValues[key])
	}
	canonicalHeaders := strings.Join(headerKeyVals, "\n")

	// get signed headers
	signedHeaders := strings.Join(headerKeys, ";")
	credentialScope := fmt.Sprintf("%s/%s/%s/%s", r.amzDate, r.client.locale.Region(), serviceName, aws4Request)
	r.headers = &headers{
		keys:             headerKeys,
		values:           headerValues,
		canonicalHeaders: canonicalHeaders,
		signedHeaders:    signedHeaders,
		credentialScope:  credentialScope,
	}
}

// canonicalRequestDigest needed for signing
// reference: https://docs.aws.amazon.com/general/latest/gr/sigv4-create-canonical-request.html
func (r *request) canonicalRequestDigest() string {
	canonicalHeaders := r.headers.canonicalHeaders
	signedHeaders := r.headers.signedHeaders
	request := []string{"POST", r.operation.GetPath(), "", canonicalHeaders, "", signedHeaders, hashedString(r.payload)}
	// canonical request string
	canonicalRequest := strings.Join(request, "\n")
	digest := hashedString([]byte(canonicalRequest))
	return digest
}

// stringToSign - creates string to sign
// reference: https://docs.aws.amazon.com/general/latest/gr/sigv4-create-string-to-sign.html
func (r *request) stringToSign() string {
	return strings.Join(
		[]string{
			authAlgorithm,
			r.amzDateTime,
			r.headers.credentialScope,
			r.canonicalRequestDigest(),
		},
		"\n",
	)
}

// sign - create signature
// reference: https://docs.aws.amazon.com/general/latest/gr/sigv4-calculate-signature.html
func (r *request) signature() string {
	// 1. calculate signing key
	kSecret := r.client.secretKey
	kDate := r.hmacSHA256([]byte("AWS4"+kSecret), r.amzDate)
	kRegion := r.hmacSHA256(kDate, r.client.locale.Region())
	kService := r.hmacSHA256(kRegion, serviceName)
	kSigning := r.hmacSHA256(kService, aws4Request)

	// 2. calculate signature
	signature := hex.EncodeToString(r.hmacSHA256(kSigning, r.stringToSign()))
	return signature
}

// authorization - get authorization header
// reference: https://docs.aws.amazon.com/general/latest/gr/sigv4-add-signature-to-request.html
func (r *request) authorization() string {
	return strings.Join(
		[]string{
			authAlgorithm,
			fmt.Sprintf("Credential=%s/%s", r.client.accessKey, r.headers.credentialScope),
			fmt.Sprintf("SignedHeaders=%s", r.headers.signedHeaders),
			fmt.Sprintf("Signature=%s", r.signature()),
		},
		" ",
	)
}

// sign - calculate signature and adds it to the header
func (r *request) sign() *request {
	if r.error != nil {
		return r
	}
	r.httpRequest.Header.Set("authorization", r.authorization())
	return r
}

// send - send request
func (r *request) send(client *http.Client) (resp []byte, err error) {
	r.httpRequest.Header.Set("accept", headerAccept)
	response, err := client.Do(r.httpRequest)
	if err != nil {
		err = fmt.Errorf("REQUEST_ERR: %s", err.Error())
		return
	}
	defer response.Body.Close()

	resp, err = ioutil.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("BODY_PARSE_ERR: %v", err)
		return
	}
	return
}

// hmacSHA256 - calculate HMAC-SHA256
func (r *request) hmacSHA256(key []byte, data string) []byte {
	hasher := hmac.New(sha256.New, key)
	_, err := hasher.Write([]byte(data))
	if err != nil {
		r.error = err
		return []byte{}
	}
	return hasher.Sum(nil)
}
