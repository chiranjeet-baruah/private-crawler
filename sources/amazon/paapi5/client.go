package paapi5

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Semantics3/go-crawler/sources/amazon/paapi5/types"
)

type (
	// Client represents accesskey needed to make request to amazon
	Client struct {
		associateTag string
		accessKey    string
		secretKey    string
		locale       types.Locale
		timeout      time.Duration
	}
)

const (
	defaultPartnerType string = "Associates"
)

// NewClient - New Amazon API client
func NewClient() (*Client, string, error) {
	return &Client{}, "", nil
}

// SetLocale - set timeout of request
func (c *Client) SetLocale(locale types.Locale) (code string, err error) {
	c.locale = locale
	c.associateTag, err = locale.AssociateTag()
	if err != nil {
		return "AMAZON_ASSOCIATE_TAG_NOT_FOUND", err
	}
	c.accessKey, err = locale.AccessKey()
	if err != nil {
		return "AMAZON_ACCESS_KEY_NOT_FOUND", err
	}
	c.secretKey, err = locale.SecretKey()
	if err != nil {
		return "AMAZON_SECRET_KEY_NOT_FOUND", err
	}
	return
}

// SetTimeout - set timeout of request
func (c *Client) SetTimeout(seconds int) {
	c.timeout = time.Second * time.Duration(seconds)
}

// makeRequest - make request to amazon and get data
func (c *Client) makeRequest(ctx context.Context, operation types.Operation, payload map[string]interface{}) (resp []byte, err error) {
	request, err := c.newRequest(ctx, operation, payload)
	if err != nil {
		return
	}

	client := &http.Client{
		Timeout: c.timeout,
	}
	resp, err = request.build().sign().send(client)
	return
}

// newRequest - build http request
func (c *Client) newRequest(ctx context.Context, operation types.Operation, payload map[string]interface{}) (*request, error) {
	// Set common payload values
	payload["PartnerType"] = defaultPartnerType
	payload["PartnerTag"] = c.associateTag
	payload["Marketplace"] = c.locale.MarketPlace()

	// Convert payload to json format
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &request{
		payload:   jsonBody,
		client:    c,
		operation: operation,
		dateTime:  time.Now().UTC(),
		ctx:       ctx,
	}, nil
}
