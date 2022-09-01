package types

import "fmt"

// Condition - custom type for condition field in GetItems API
type Condition string

// Condition enums available
const (
	Any         Condition = "Any"
	New         Condition = "New"
	Used        Condition = "Used"
	Collectible Condition = "Collectible"
	Refurbished Condition = "Refurbished"
)

// Currency custom type for currencies
type Currency string

const (
	// USD United States Dollar
	USD Currency = "USD"
	// EUR Euro
	EUR Currency = "EUR"
	// GBP British Pound
	GBP Currency = "GBP"
)

// Language custom type for languages
type Language string

const (
	// EnglishUSA English (United States)
	EnglishUSA Language = "en_US"
	// EnglishUK English (United Kingdom)
	EnglishUK Language = "en_GB"
)

// Merchant Enum for merchant types
type Merchant string

const (
	// AllMerchants any merchant
	AllMerchants Merchant = "All"
	// Amazon Amazon merchant
	Amazon Merchant = "Amazon"
)

// Resource determine what information will be returned in the API response
// Reference for resources https://webservices.amazon.com/paapi5/documentation/get-items.html#resources-parameter
type Resource string

// Constants represents enum values for Resources
const (
	BrowseNodesAncestor                                    Resource = "BrowseNodes.Ancestor"
	BrowseNodesChildren                                    Resource = "BrowseNodes.Children"
	BrowseNodeInfoBrowseNodes                              Resource = "BrowseNodeInfo.BrowseNodes"
	BrowseNodeInfoBrowseNodesAncestor                      Resource = "BrowseNodeInfo.BrowseNodes.Ancestor"
	BrowseNodeInfoBrowseNodesSalesRank                     Resource = "BrowseNodeInfo.BrowseNodes.SalesRank"
	BrowseNodeInfoWebsiteSalesRank                         Resource = "BrowseNodeInfo.WebsiteSalesRank"
	CustomerReviewsCount                                   Resource = "CustomerReviews.Count"
	CustomerReviewsStarRating                              Resource = "CustomerReviews.StarRating"
	ImagesPrimarySmall                                     Resource = "Images.Primary.Small"
	ImagesPrimaryMedium                                    Resource = "Images.Primary.Medium"
	ImagesPrimaryLarge                                     Resource = "Images.Primary.Large"
	ImagesVariantsSmall                                    Resource = "Images.Variants.Small"
	ImagesVariantsMedium                                   Resource = "Images.Variants.Medium"
	ImagesVariantsLarge                                    Resource = "Images.Variants.Large"
	ItemInfoByLineInfo                                     Resource = "ItemInfo.ByLineInfo"
	ItemInfoContentInfo                                    Resource = "ItemInfo.ContentInfo"
	ItemInfoContentRating                                  Resource = "ItemInfo.ContentRating"
	ItemInfoClassifications                                Resource = "ItemInfo.Classifications"
	ItemInfoExternalIds                                    Resource = "ItemInfo.ExternalIds"
	ItemInfoFeatures                                       Resource = "ItemInfo.Features"
	ItemInfoManufactureInfo                                Resource = "ItemInfo.ManufactureInfo"
	ItemInfoProductInfo                                    Resource = "ItemInfo.ProductInfo"
	ItemInfoTechnicalInfo                                  Resource = "ItemInfo.TechnicalInfo"
	ItemInfoTitle                                          Resource = "ItemInfo.Title"
	ItemInfoTradeInInfo                                    Resource = "ItemInfo.TradeInInfo"
	OffersListingsAvailabilityMaxOrderQuantity             Resource = "Offers.Listings.Availability.MaxOrderQuantity"
	OffersListingsAvailabilityMessage                      Resource = "Offers.Listings.Availability.Message"
	OffersListingsAvailabilityMinOrderQuantity             Resource = "Offers.Listings.Availability.MinOrderQuantity"
	OffersListingsAvailabilityType                         Resource = "Offers.Listings.Availability.Type"
	OffersListingsCondition                                Resource = "Offers.Listings.Condition"
	OffersListingsConditionConditionNote                   Resource = "Offers.Listings.Condition.ConditionNote"
	OffersListingsConditionSubCondition                    Resource = "Offers.Listings.Condition.SubCondition"
	OffersListingsDeliveryInfoIsAmazonFulfilled            Resource = "Offers.Listings.DeliveryInfo.IsAmazonFulfilled"
	OffersListingsDeliveryInfoIsFreeShippingEligible       Resource = "Offers.Listings.DeliveryInfo.IsFreeShippingEligible"
	OffersListingsDeliveryInfoIsPrimeEligible              Resource = "Offers.Listings.DeliveryInfo.IsPrimeEligible"
	OffersListingsDeliveryInfoShippingCharges              Resource = "Offers.Listings.DeliveryInfo.ShippingCharges"
	OffersListingsIsBuyBoxWinner                           Resource = "Offers.Listings.IsBuyBoxWinner"
	OffersListingsLoyaltyPointsPoints                      Resource = "Offers.Listings.LoyaltyPoints.Points"
	OffersListingsMerchantInfo                             Resource = "Offers.Listings.MerchantInfo"
	OffersListingsPrice                                    Resource = "Offers.Listings.Price"
	OffersListingsProgramEligibilityIsPrimeExclusive       Resource = "Offers.Listings.ProgramEligibility.IsPrimeExclusive"
	OffersListingsProgramEligibilityIsPrimePantry          Resource = "Offers.Listings.ProgramEligibility.IsPrimePantry"
	OffersListingsPromotions                               Resource = "Offers.Listings.Promotions"
	OffersListingsSavingBasis                              Resource = "Offers.Listings.SavingBasis"
	OffersSummariesHighestPrice                            Resource = "Offers.Summaries.HighestPrice"
	OffersSummariesLowestPrice                             Resource = "Offers.Summaries.LowestPrice"
	OffersSummariesOfferCount                              Resource = "Offers.Summaries.OfferCount"
	ParentASIN                                             Resource = "ParentASIN"
	RentalOffersListingsAvailabilityMaxOrderQuantity       Resource = "RentalOffers.Listings.Availability.MaxOrderQuantity"
	RentalOffersListingsAvailabilityMessage                Resource = "RentalOffers.Listings.Availability.Message"
	RentalOffersListingsAvailabilityMinOrderQuantity       Resource = "RentalOffers.Listings.Availability.MinOrderQuantity"
	RentalOffersListingsAvailabilityType                   Resource = "RentalOffers.Listings.Availability.Type"
	RentalOffersListingsBasePrice                          Resource = "RentalOffers.Listings.BasePrice"
	RentalOffersListingsCondition                          Resource = "RentalOffers.Listings.Condition"
	RentalOffersListingsConditionSubCondition              Resource = "RentalOffers.Listings.Condition.SubCondition"
	RentalOffersListingsDeliveryInfoIsAmazonFulfilled      Resource = "RentalOffers.Listings.DeliveryInfo.IsAmazonFulfilled"
	RentalOffersListingsDeliveryInfoIsFreeShippingEligible Resource = "RentalOffers.Listings.DeliveryInfo.IsFreeShippingEligible"
	RentalOffersListingsDeliveryInfoIsPrimeEligible        Resource = "RentalOffers.Listings.DeliveryInfo.IsPrimeEligible"
	RentalOffersListingsDeliveryInfoShippingCharges        Resource = "RentalOffers.Listings.DeliveryInfo.ShippingCharges"
	RentalOffersListingsMerchantInfo                       Resource = "RentalOffers.Listings.MerchantInfo"
	VariationSummaryPriceHighestPrice                      Resource = "VariationSummary.Price.HighestPrice"
	VariationSummaryPriceLowestPrice                       Resource = "VariationSummary.Price.LowestPrice"
	VariationSummaryVariationDimension                     Resource = "VariationSummary.VariationDimension"
	SearchRefinements                                      Resource = "SearchRefinements"
)

// Error represents Error object in json response
type Error struct {
	Type    string `json:"__type,omitempty"`
	Code    string `json:"Code,omitempty"`
	Message string `json:"Message,omitempty"`
}

// ToError converts amazon error to error format
func (e Error) ToError() (string, error) {
	code := normalizeErrors(e.Message)
	return code, fmt.Errorf("%s, Error Code: %s, Error Type: %s", e.Message, e.Code, e.Type)
}

// ItemsResult represents ItemsResult object in json response
type ItemsResult struct {
	Items []Item `json:"Items,omitempty"`
}

// ItemInfo represents ItemInfo object in json response
type ItemInfo struct {
	ByLineInfo      ByLineInfo                  `json:"ByLineInfo,omitempty"`
	Classifications Classifications             `json:"Classifications,omitempty"`
	ContentInfo     ContentInfo                 `json:"ContentInfo,omitempty"`
	ContentRating   ContentRating               `json:"ContentRating,omitempty"`
	ExternalIds     ExternalIds                 `json:"ExternalIds,omitempty"`
	Features        MultiValuedAttribute        `json:"Features,omitempty"`
	ManufactureInfo ManufactureInfo             `json:"ManufactureInfo,omitempty"`
	ProductInfo     ProductInfo                 `json:"ProductInfo,omitempty"`
	TechnicalInfo   TechnicalInfo               `json:"TechnicalInfo,omitempty"`
	Title           SingleStringValuedAttribute `json:"Title,omitempty"`
	TradeInInfo     TradeInInfo                 `json:"TradeInInfo,omitempty"`
}

// RentalOffers represents RentalOffers object in json response
type RentalOffers struct {
	Listings []RentalOfferListing `json:"Listings,omitempty"`
}

// ContentInfo represents ContentInfo object in json response
type ContentInfo struct {
	Edition         SingleStringValuedAttribute  `json:"Edition,omitempty"`
	Languages       Languages                    `json:"Languages,omitempty"`
	PagesCount      SingleIntegerValuedAttribute `json:"PagesCount,omitempty"`
	PublicationDate SingleStringValuedAttribute  `json:"PublicationDate,omitempty"`
}

// ExternalIds represents ExternalIds object in json response
type ExternalIds struct {
	EANs  MultiValuedAttribute `json:"EANs,omitempty"`
	ISBNs MultiValuedAttribute `json:"ISBNs,omitempty"`
	UPCs  MultiValuedAttribute `json:"UPCs,omitempty"`
}

// TechnicalInfo represents TechnicalInfo object in json response
type TechnicalInfo struct {
	Formats MultiValuedAttribute `json:"Formats,omitempty"`
}

// TradeInInfo represents TradeInInfo object in json response
type TradeInInfo struct {
	IsEligibleForTradeIn bool         `json:"IsEligibleForTradeIn,omitempty"`
	Price                TradeInPrice `json:"Price,omitempty"`
}

// TradeInPrice represents TradeInPrice object in json response
type TradeInPrice struct {
	Amount        float32 `json:"Amount,omitempty"`
	Currency      string  `json:"Currency,omitempty"`
	DisplayAmount string  `json:"DisplayAmount,omitempty"`
}

// SingleIntegerValuedAttribute represents SingleIntegerValuedAttribute object in json response
type SingleIntegerValuedAttribute struct {
	DisplayValue int    `json:"DisplayValue,omitempty"`
	Label        string `json:"Label,omitempty"`
	Locale       string `json:"Locale,omitempty"`
}

// MultiValuedAttribute represents MultiValuedAttribute object in json response
type MultiValuedAttribute struct {
	DisplayValues []string `json:"DisplayValues,omitempty"`
	Label         string   `json:"Label,omitempty"`
	Locale        string   `json:"Locale,omitempty"`
}

// Languages represents Languages object in json response
type Languages struct {
	DisplayValues []LanguageType `json:"DisplayValues,omitempty"`
	Label         string         `json:"Label,omitempty"`
	Locale        string         `json:"Locale,omitempty"`
}

// LanguageType represents LanguageType object in json response
type LanguageType struct {
	DisplayValue string `json:"DisplayValue,omitempty"`
	Type         string `json:"Type,omitempty"`
}

// ManufactureInfo represents ManufactureInfo object in json response
type ManufactureInfo struct {
	ItemPartNumber SingleStringValuedAttribute `json:"ItemPartNumber,omitempty"`
	Model          SingleStringValuedAttribute `json:"Model,omitempty"`
	Warranty       SingleStringValuedAttribute `json:"Warranty,omitempty"`
}

// ProductInfo represents ProductInfo object in json response
type ProductInfo struct {
	Color          SingleStringValuedAttribute  `json:"Color,omitempty"`
	IsAdultProduct SingleBooleanValuedAttribute `json:"IsAdultProduct,omitempty"`
	ItemDimensions DimensionBasedAttribute      `json:"ItemDimensions,omitempty"`
	ReleaseDate    SingleStringValuedAttribute  `json:"ReleaseDate,omitempty"`
	Size           SingleStringValuedAttribute  `json:"Size,omitempty"`
	UnitCount      SingleIntegerValuedAttribute `json:"UnitCount,omitempty"`
}

// SingleBooleanValuedAttribute represents SingleBooleanValuedAttribute object in json response
type SingleBooleanValuedAttribute struct {
	DisplayValue bool   `json:"DisplayValue,omitempty"`
	Label        string `json:"Label,omitempty"`
	Locale       string `json:"Locale,omitempty"`
}

// DimensionBasedAttribute represents DimensionBasedAttribute object in json response
type DimensionBasedAttribute struct {
	Height UnitBasedAttribute `json:"Height,omitempty"`
	Length UnitBasedAttribute `json:"Length,omitempty"`
	Weight UnitBasedAttribute `json:"Weight,omitempty"`
	Width  UnitBasedAttribute `json:"Width,omitempty"`
}

func (d DimensionBasedAttribute) GetDimension() (res string) {
	if d.Length.DisplayValue == 0 || d.Width.DisplayValue == 0 || d.Height.DisplayValue == 0 {
		return
	}
	return fmt.Sprintf("%v x %v x %v %s", d.Length.DisplayValue, d.Width.DisplayValue, d.Height.DisplayValue, d.Height.Unit)
}

// UnitBasedAttribute represents UnitBasedAttribute object in json response
type UnitBasedAttribute struct {
	DisplayValue float32 `json:"DisplayValue,omitempty"`
	Label        string  `json:"Label,omitempty"`
	Locale       string  `json:"Locale,omitempty"`
	Unit         string  `json:"Unit,omitempty"`
}

// ContentRating represents ContentRating object in json response
type ContentRating struct {
	AudienceRating SingleStringValuedAttribute `json:"AudienceRating,omitempty"`
}

// Classifications represents Classifications object in json response
type Classifications struct {
	Binding      SingleStringValuedAttribute `json:"Binding,omitempty"`
	ProductGroup SingleStringValuedAttribute `json:"ProductGroup,omitempty"`
}

// VariationAttribute represents VariationAttribute object in json response
type VariationAttribute struct {
	Name  string `json:"Name,omitempty"`
	Value string `json:"Value,omitempty"`
}

// RentalOfferListing represents RentalOfferListing object in json response
type RentalOfferListing struct {
	Availability OfferAvailability `json:"Availability,omitempty"`
	BasePrice    DurationPrice     `json:"BasePrice,omitempty"`
	Condition    OfferCondition    `json:"Condition,omitempty"`
	DeliveryInfo OfferDeliveryInfo `json:"DeliveryInfo,omitempty"`
	ID           string            `json:"Id,omitempty"`
	MerchantInfo OfferMerchantInfo `json:"MerchantInfo,omitempty"`
}

// DurationPrice represents DurationPrice object in json response
type DurationPrice struct {
	Price    OfferPrice         `json:"Price,omitempty"`
	Duration UnitBasedAttribute `json:"Duration,omitempty"`
}

// ByLineInfo represents ByLineInfo object in json response
type ByLineInfo struct {
	Brand        SingleStringValuedAttribute `json:"Brand,omitempty"`
	Contributors []Contributor               `json:"Contributors,omitempty"`
	Manufacturer SingleStringValuedAttribute `json:"Manufacturer,omitempty"`
}

// SingleStringValuedAttribute represents SingleStringValuedAttribute object in json response
type SingleStringValuedAttribute struct {
	DisplayValue string `json:"DisplayValue,omitempty"`
	Label        string `json:"Label,omitempty"`
	Locale       string `json:"Locale,omitempty"`
}

// Contributor represents Contributor object in json response
type Contributor struct {
	Locale string `json:"Locale,omitempty"`
	Name   string `json:"Name,omitempty"`
	Role   string `json:"Role,omitempty"`
}
