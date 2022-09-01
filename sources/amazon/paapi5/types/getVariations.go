package types

import (
	"errors"
)

// GetItemsParams holds parameters to be passed to GetItems operation
// reference: https://webservices.amazon.com/paapi5/documentation/get-items.html#ItemLookup-rp
type (
	GetVariationsParams struct {
		// ASIN required: yes, e.g. "B0199980K4"
		ASIN string
		// Condition - required: no, default: Any
		Condition Condition
		// CurrencyOfPreference - required: no, default: None
		CurrencyOfPreference Currency
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
		// VariationCount required: no, default: 10
		VariationCount int
		// VariationPage required: no, default: 1
		VariationPage int
	}

	// GetItemsResponse holds response from GetItems operation
	// Reference https://webservices.amazon.com/paapi5/documentation/get-items.html#ItemLookup-sr
	GetVariationsResponse struct {
		Errors           []Error          `json:"Errors,omitempty"`
		VariationsResult VariationsResult `json:"VariationsResult,omitempty"`
		request          *GetVariationsParams
	}

	// VariationsResult represents VariationsResult object in json response
	VariationsResult struct {
		Items            []Item           `json:"Items,omitempty"`
		VariationSummary VariationSummary `json:"VariationSummary,omitempty"`
	}

	// VariationSummary represents VariationSummary object in json response
	VariationSummary struct {
		PageCount           int                  `json:"PageCount,omitempty"`
		Price               Price                `json:"Price,omitempty"`
		VariationCount      int                  `json:"VariationCount,omitempty"`
		VariationDimensions []VariationDimension `json:"VariationDimensions,omitempty"`
	}

	// Price represents Price object in json response
	Price struct {
		HighestPrice OfferPrice `json:"HighestPrice,omitempty"`
		LowestPrice  OfferPrice `json:"LowestPrice,omitempty"`
	}

	// VariationDimension represents VariationDimension object in json response
	VariationDimension struct {
		DisplayName string   `json:"DisplayName,omitempty"`
		Locale      string   `json:"Locale,omitempty"`
		Name        string   `json:"Name,omitempty"`
		Values      []string `json:"Values,omitempty"`
	}
)

// Payload outputs payload map
func (p *GetVariationsParams) Payload() (res map[string]interface{}, err error) {
	res = make(map[string]interface{})
	if p.ASIN != "" {
		res["ASIN"] = p.ASIN
	} else {
		return nil, errors.New("ASIN is required")
	}

	if p.Condition != "" {
		res["Condition"] = p.Condition
	}

	if p.CurrencyOfPreference != "" {
		res["CurrencyOfPreference"] = p.CurrencyOfPreference
	}

	if len(p.LanguagesOfPreference) > 0 {
		res["LanguagesOfPreference"] = p.LanguagesOfPreference
	}

	if p.Merchant != "" {
		res["Merchant"] = p.Merchant
	}

	if p.OfferCount > 0 {
		res["OfferCount"] = p.OfferCount
	}

	if len(p.Resources) > 0 {
		res["Resources"] = p.Resources
	}

	if p.VariationCount > 0 {
		res["VariationCount"] = p.VariationCount
	}

	if p.VariationPage > 0 {
		res["VariationPage"] = p.VariationPage
	}

	return res, nil
}

// Normalized - normalizes the data in sem3 format
func (r *GetVariationsResponse) Normalized(geoID int) (res []map[string]interface{}, code string, err error) {
	if len(r.Errors) > 0 {
		code, err = r.Errors[0].ToError()
		return
	}
	items := r.VariationsResult.Items
	if len(items) < 1 {
		return
	}
	res = make([]map[string]interface{}, len(items))
	for idx, item := range items {
		res[idx] = item.Normalize(geoID)
		// We need the queried ASIN to be the first element of res array
		if item.ASIN == r.request.ASIN && idx != 0 {
			temp := res[0]
			res[0] = res[idx]
			res[idx] = temp
		}
	}
	return
}

func (r *GetVariationsResponse) SetRequestParams(request *GetVariationsParams) {
	r.request = request
}
