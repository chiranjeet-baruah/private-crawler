package types

import (
	"errors"
	"log"
	"regexp"
)

// GetItemsParams holds parameters to be passed to GetItems operation
// reference: https://webservices.amazon.com/paapi5/documentation/get-items.html#ItemLookup-rp
type (
	GetItemsParams struct {
		// Condition - required: no, default: Any
		Condition Condition
		// CurrencyOfPreference - required: no, default: None
		CurrencyOfPreference Currency
		// ItemIds required: yes, e.p. ["B0199980K4", "B000HZD168"]
		ItemIds []string
		// LanguagesOfPreference required: no, default: None
		LanguagesOfPreference []Language
		// Marketplace required: no, default: None, e.p. "www.amazon.com"
		Marketplace string
		// Merchant required: no, default: All
		Merchant Merchant
		// OfferCount required: no, default: 1
		OfferCount int
		// Resources required: no, default: ["ItemInfo.Title"]
		Resources []Resource
	}

	// GetItemsResponse holds response from GetItems operation
	// Reference https://webservices.amazon.com/paapi5/documentation/get-items.html#ItemLookup-sr
	GetItemsResponse struct {
		Errors      []Error     `json:"Errors,omitempty"`
		ItemsResult ItemsResult `json:"ItemsResult,omitempty"`
	}

	// ItemResponse represents response for a single asin in GetItems
	ItemResponse struct {
		Data  []map[string]interface{}
		Code  string
		Error error
	}
)

func NewMultiItemResponse(arrLen int, code string, err error) []*ItemResponse {
	res := make([]*ItemResponse, arrLen)
	for i := 0; i < arrLen; i++ {
		res[i] = NewItemResponse(nil, code, err)
	}
	return res
}

func NewItemResponse(data []map[string]interface{}, code string, err error) *ItemResponse {
	return &ItemResponse{
		Data:  data,
		Code:  code,
		Error: err,
	}
}

// asinFromError get product ASIN from error message
func asinFromError(message string) string {
	rgx := `ItemId\s([A-Z0-9]{10})`
	Re := regexp.MustCompile(rgx)
	capturedVals := Re.FindStringSubmatch(message)

	if len(capturedVals) < 2 {
		return ""
	} else if len(capturedVals[1]) < 10 {
		return ""
	}
	return capturedVals[1]
}

// Payload outputs payload map
func (p *GetItemsParams) Payload() (res map[string]interface{}, err error) {
	res = make(map[string]interface{})
	res["ItemIdType"] = "ASIN"

	if p.Condition != "" {
		res["Condition"] = p.Condition
	}

	if p.CurrencyOfPreference != "" {
		res["CurrencyOfPreference"] = p.CurrencyOfPreference
	}

	if len(p.ItemIds) > 0 {
		res["ItemIds"] = p.ItemIds
	} else {
		return nil, errors.New("atleast one item id is required")
	}

	if len(p.LanguagesOfPreference) > 0 {
		res["LanguagesOfPreference"] = p.LanguagesOfPreference
	}

	if p.Merchant != "" {
		res["Merchant"] = p.Merchant
	}

	if p.OfferCount > 1 {
		res["OfferCount"] = p.OfferCount
	}

	if len(p.Resources) > 0 {
		res["Resources"] = p.Resources
	}

	return res, nil
}

// Normalized - normalizes the data in sem3 format
// TODO - will take single value at the time, update as required
func (r *GetItemsResponse) Normalized(geoID int, asins []string) []*ItemResponse {
	// normalize and add item responses to asin map
	resMap := make(map[string]*ItemResponse, len(asins))
	items := r.ItemsResult.Items
	for _, item := range items {
		normalizedData := item.Normalize(geoID)
		resMap[item.ASIN] = &ItemResponse{[]map[string]interface{}{normalizedData}, "", nil}
	}

	// add errors to asin map
	for _, amzErr := range r.Errors {
		asin := asinFromError(amzErr.Message)
		code, err := amzErr.ToError()
		if asin == "" {
			log.Printf("AMAZON_ERR: %s, %v", code, err)
		} else {
			resMap[asin] = &ItemResponse{nil, code, err}
		}
	}

	res := make([]*ItemResponse, len(asins))
	for i := 0; i < len(asins); i++ {
		if resMap[asins[i]] != nil {
			res[i] = resMap[asins[i]]
		} else {
			res[i] = &ItemResponse{nil, "AMAZON_NORMALIZE_ERROR", errors.New("no data or error found")}
		}
	}

	return res
}
