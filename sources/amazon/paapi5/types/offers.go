package types

import (
	"fmt"
	"strings"
)

// Offers represents Offers object in json response
type Offers struct {
	Listings  []OfferListing `json:"Listings,omitempty"`
	Summaries []OfferSummary `json:"Summaries,omitempty"`
}

func (o Offers) GetOffers() []map[string]string {
	listings := o.Listings
	res := make([]map[string]string, len(listings))
	for idx, listing := range listings {
		res[idx] = listing.GetOffer()
	}
	return res
}

// OfferListing represents OfferListing object in json response
type OfferListing struct {
	Availability       OfferAvailability       `json:"Availability,omitempty"`
	Condition          OfferCondition          `json:"Condition,omitempty"`
	DeliveryInfo       OfferDeliveryInfo       `json:"DeliveryInfo,omitempty"`
	ID                 string                  `json:"Id,omitempty"`
	IsBuyBoxWinner     bool                    `json:"IsBuyBoxWinner,omitempty"`
	LoyaltyPoints      OfferLoyaltyPoints      `json:"LoyaltyPoints,omitempty"`
	MerchantInfo       OfferMerchantInfo       `json:"MerchantInfo,omitempty"`
	Price              OfferPrice              `json:"Price,omitempty"`
	ProgramEligibility OfferProgramEligibility `json:"ProgramEligibility,omitempty"`
	Promotions         []OfferPromotion        `json:"Promotions,omitempty"`
	SavingBasis        OfferPrice              `json:"SavingBasis,omitempty"`
	ViolatesMAP        bool                    `json:"ViolatesMAP,omitempty"`
}

func (o OfferListing) GetOffer() (res map[string]string) {
	res = make(map[string]string)
	availabilityMessage := strings.ToLower(o.Availability.Message)
	normalizedAvailabilityMessage := normalizeAvailability(availabilityMessage)

	// Get Basic offer fields when the product irrespective of it availability status
	if o.MerchantInfo.Name != "" {
		res["seller"] = o.MerchantInfo.Name
	}

	if o.Price.Currency != "" {
		res["currency"] = o.Price.Currency
	}

	if o.Price.Amount != 0 {
		res["price"] = fmt.Sprintf("%v", o.Price.Amount)
	}

	if o.Condition.DisplayValue != "" {
		res["condition"] = o.Condition.DisplayValue
	}

	if normalizedAvailabilityMessage != "in stock." {
		res["is_available"] = "0"

		availability := "Not Available"
		if o.Availability.Message != "" {
			unavailableMessage := "[" + o.Availability.Message + "]"
			availability = fmt.Sprintf("%s %s", availability, unavailableMessage)
		}

		res["availability"] = availability
		res["availability_raw"] = o.Availability.Message

		return
	}

	res["availability_raw"] = o.Availability.Message
	res["is_available"] = "1"
	availability := "Available"
	buyBox := ""
	amazonFulfilled := ""
	freeShipping := ""

	if o.IsBuyBoxWinner {
		buyBox = "[BBX: Buy Box]"
	}
	if o.DeliveryInfo.IsAmazonFulfilled {
		amazonFulfilled = "[FBA: Fulfilled by Amazon]"
	}
	if o.DeliveryInfo.IsPrimeEligible || o.DeliveryInfo.IsFreeShippingEligible {
		freeShipping = "[APR: Shipping with Amazon Prime]"
	}

	if o.IsBuyBoxWinner || o.DeliveryInfo.IsAmazonFulfilled || o.DeliveryInfo.IsPrimeEligible || o.DeliveryInfo.IsFreeShippingEligible {
		availability = fmt.Sprintf("%s %s%s%s", availability, buyBox, amazonFulfilled, freeShipping)
	}
	res["availability"] = availability

	return res
}

// type availabilityMessage string
// func (o availabilityMessage) CheckAvailability() (res string) {

// }

//Normalize different regional availability strings
func normalizeAvailability(message string) (res string) {
	if message == "in stock." {
		res = "in stock."
	}

	if strings.Contains(message, "in stock. usually ships") {
		res = "in stock."
	}

	if strings.Contains(message, "left in stock") {
		res = "in stock."
	}

	if strings.Contains(message, "in stock soon.") {
		res = "in stock."
	}

	if message == "disponibilità immediata." {
		res = "in stock."
	}
	if message == "en stock." {
		res = "in stock."
	}
	if message == "在庫あり。" {
		res = "in stock."
	}

	return res
}

// OfferAvailability represents OfferAvailability object in json response
type OfferAvailability struct {
	MaxOrderQuantity int    `json:"MaxOrderQuantity,omitempty"`
	Message          string `json:"Message,omitempty"`
	MinOrderQuantity int    `json:"MinOrderQuantity,omitempty"`
	Type             string `json:"Type,omitempty"`
}

// OfferCondition represents OfferCondition object in json response
type OfferCondition struct {
	DisplayValue string            `json:"DisplayValue,omitempty"`
	Label        string            `json:"Label,omitempty"`
	Locale       string            `json:"Locale,omitempty"`
	Value        string            `json:"Value,omitempty"`
	SubCondition OfferSubCondition `json:"SubCondition,omitempty"`
}

// OfferMerchantInfo represents OfferMerchantInfo object in json response
type OfferMerchantInfo struct {
	DefaultShippingCountry string `json:"DefaultShippingCountry,omitempty"`
	ID                     string `json:"Id,omitempty"`
	Name                   string `json:"Name,omitempty"`
}

// OfferSubCondition represents OfferSubCondition object in json response
type OfferSubCondition struct {
	DisplayValue string `json:"DisplayValue,omitempty"`
	Label        string `json:"Label,omitempty"`
	Locale       string `json:"Locale,omitempty"`
	Value        string `json:"Value,omitempty"`
}

// OfferDeliveryInfo represents OfferDeliveryInfo object in json response
type OfferDeliveryInfo struct {
	IsAmazonFulfilled      bool                  `json:"IsAmazonFulfilled,omitempty"`
	IsFreeShippingEligible bool                  `json:"IsFreeShippingEligible,omitempty"`
	IsPrimeEligible        bool                  `json:"IsPrimeEligible,omitempty"`
	ShippingCharges        []OfferShippingCharge `json:"ShippingCharges,omitempty"`
}

// OfferShippingCharge represents OfferShippingCharge object in json response
type OfferShippingCharge struct {
	Amount             float32 `json:"Amount,omitempty"`
	Currency           string  `json:"Currency,omitempty"`
	DisplayAmount      string  `json:"DisplayAmount,omitempty"`
	IsRateTaxInclusive bool    `json:"IsRateTaxInclusive,omitempty"`
	Type               string  `json:"Type,omitempty"`
}

// OfferSummary represents OfferSummary object in json response
type OfferSummary struct {
	Condition    OfferCondition `json:"Condition,omitempty"`
	HighestPrice OfferPrice     `json:"HighestPrice,omitempty"`
	LowestPrice  OfferPrice     `json:"LowestPrice,omitempty"`
	OfferCount   int            `json:"OfferCount,omitempty"`
}

// OfferPrice represents OfferPrice object in json response
type OfferPrice struct {
	Amount        float32      `json:"Amount,omitempty"`
	Currency      string       `json:"Currency,omitempty"`
	DisplayAmount string       `json:"DisplayAmount,omitempty"`
	PricePerUnit  float32      `json:"PricePerUnit,omitempty"`
	Savings       OfferSavings `json:"Savings,omitempty"`
}

// OfferLoyaltyPoints represents OfferLoyaltyPoints object in json response
type OfferLoyaltyPoints struct {
	Points int `json:"Points,omitempty"`
}

// OfferPromotion represents OfferPromotion object in json response
type OfferPromotion struct {
	Amount          float32 `json:"Amount,omitempty"`
	Currency        string  `json:"Currency,omitempty"`
	DiscountPercent int     `json:"DiscountPercent,omitempty"`
	DisplayAmount   string  `json:"DisplayAmount,omitempty"`
	PricePerUnit    float32 `json:"PricePerUnit,omitempty"`
	Type            string  `json:"Type,omitempty"`
}

// OfferProgramEligibility represents OfferProgramEligibility object in json response
type OfferProgramEligibility struct {
	IsPrimeExclusive bool `json:"IsPrimeExclusive,omitempty"`
	IsPrimePantry    bool `json:"IsPrimePantry,omitempty"`
}

// OfferSavings represents OfferSavings object in json response
type OfferSavings struct {
	Amount        float32 `json:"Amount,omitempty"`
	Currency      string  `json:"Currency,omitempty"`
	DisplayAmount string  `json:"DisplayAmount,omitempty"`
	Percentage    int     `json:"Percentage,omitempty"`
	PricePerUnit  float32 `json:"PricePerUnit,omitempty"`
}
