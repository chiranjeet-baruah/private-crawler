package m101

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

// M101 - implements Source interface and
type M101Client struct {
	apiKey      string
	apiEndpoint string
}

func NewM101Client() (c *M101Client, code string, err error) {
	apiKey := os.Getenv("M101_API_KEY")
	c = &M101Client{
		apiKey:      apiKey,
		apiEndpoint: "https://api.monetizer101.com",
	}
	return
}

// makeRequest - calls Monetizer101 endpoint
func (m *M101Client) GetOfferIDItem(offerID string) (res map[string]interface{}, err error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	endpoint := fmt.Sprintf("%s/offers-v1.0/%s", m.apiEndpoint, offerID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("REQUEST_ERR: %s", err.Error())
	}
	// set api key in header
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("X-Api-Key", m.apiKey)
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("got error %s", err.Error())
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("BODY_PARSE_ERR: %v", err)
	}
	var resp map[string]interface{}
	// fmt.Println("M101_DATA_RESPONSE:", string(body))
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("JSON_UNMARSHAL_ERR: %v, RESP_BODY: %s", err, string(body))
	}
	normalized := m.Normalize(resp)
	return normalized, nil
}

// normalize each dataset returned
func (m *M101Client) Normalize(product map[string]interface{}) (normalized map[string]interface{}) {
	offer := make(map[string]interface{})
	normalized = make(map[string]interface{})
	name, ok := cutils.GetStringKey(product, "name")
	if ok && name != "" {
		normalized["name"] = name
	} else {
		return nil
	}

	merchant, ok := cutils.GetMapInterface(product, "merchant_name")
	if ok && merchant != nil {
		offer["seller"] = merchant
	}

	lp, ok := cutils.GetFloatKey(product, "retail_price")
	if ok && lp != 0 {
		normalized["listprice"] = fmt.Sprintf("%.2f", lp)
	}
	lpc, ok := cutils.GetStringKey(product, "currency")
	if ok && lpc != "" {
		normalized["listprice_currency"] = lpc
		offer["currency"] = lpc
	}
	url, ok := cutils.GetStringKey(product, "url")
	if ok && url != "" {
		normalized["url"] = url
		if site, _ := getDomainName(url); site != "" {
			normalized["site"] = site
		} else {
			return nil
		}
	}
	brand, ok := cutils.GetStringKey(product, "brand")
	if ok && brand != "" {
		normalized["brand"] = brand
	}

	img, ok := cutils.GetStringKey(product, "image_url")
	if ok && img != "" {
		normalized["images"] = []interface{}{img}
	}

	sp, ok := cutils.GetFloatKey(product, "sale_price")
	if ok && sp != 0 {
		offer["price"] = fmt.Sprintf("%.2f", sp)
		avl, ok := cutils.GetBoolKey(product, "in_stock")
		if ok && avl {
			offer["availability"] = "Available"
		}
		normalized["offers"] = []map[string]interface{}{offer}
	}
	return
}

func (m *M101Client) GetOfferID(productURL string) (offerID string, err error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	apiKey := m.apiKey
	endpoint := m.apiEndpoint + "/offers-v1.0/"
	payload, err := json.Marshal(map[string]interface{}{
		"url":    productURL,
		"market": "usd_en",
	})
	if err != nil {
		err = fmt.Errorf("PAYLOAD_MARSHAL_ERR: %s", err.Error())
		return
	}
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload))
	if err != nil {
		err = fmt.Errorf("REQUEST_ERR: %s", err.Error())
		return
	}
	// set api key in header
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("X-Api-Key", apiKey)
	response, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("got error %s", err.Error())
		return
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("BODY_PARSE_ERR: %v", err)
		return
	}
	//fmt.Println("M101_OFFER_RESPONSE:", string(body))
	var resp map[string]interface{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		err = fmt.Errorf("JSON_UNMARSHAL_ERR: %v, RESP_BODY: %s", err, string(body))
		return
	}

	if offerID, ok := resp["offer_id"].(string); ok {
		return offerID, nil
	}
	err = fmt.Errorf("OFFER_ID_ERROR: %s", string(body))
	return
}

// As M101 changed their querying mechanism, we can get all data with just one API call instead of two
func (m *M101Client) GetProductData(productURL string) (res map[string]interface{}, err error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	apiKey := m.apiKey
	endpoint := m.apiEndpoint + "/offers-v1.0/"
	payload, err := json.Marshal(map[string]interface{}{
		"url":    productURL,
		"market": "usd_en",
	})
	if err != nil {
		err = fmt.Errorf("PAYLOAD_MARSHAL_ERR: %s", err.Error())
		return
	}
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload))
	if err != nil {
		err = fmt.Errorf("REQUEST_ERR: %s", err.Error())
		return
	}
	// set api key in header
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("X-Api-Key", apiKey)
	response, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("got error %s", err.Error())
		return
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("BODY_PARSE_ERR: %v", err)
		return
	}
	// fmt.Println("M101_SINGLE_CALL_RESPONSE:", string(body))
	var resp map[string]interface{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		err = fmt.Errorf("JSON_UNMARSHAL_ERR: %v, RESP_BODY: %s", err, string(body))
		return
	}
	normalized := m.Normalize(resp)
	return normalized, nil
}

// getDomainName from url
func getDomainName(productURL string) (domain string, err error) {
	parsedURL, err := url.Parse(productURL)
	if err != nil {
		return
	}
	host := parsedURL.Hostname()
	domain = strings.Replace(host, "www.", "", 1)
	return domain, nil
}
