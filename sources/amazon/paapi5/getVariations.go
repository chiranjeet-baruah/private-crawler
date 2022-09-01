package paapi5

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/Semantics3/go-crawler/sources/amazon/paapi5/types"
)

// GetVariations - get information for items with variations
func (c *Client) GetVariations(ctx context.Context, params *types.GetVariationsParams) (*types.GetVariationsResponse, error) {
	/* Uncomment to save and read from saved item */
	// respB, err := readFile(fmt.Sprintf("sources/amazon/paapi5/example_response/%s_variations.json", params.ASIN))
	// if err == nil {
	// 	var resp types.GetVariationsResponse
	// 	err = json.Unmarshal(respB, &resp)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("JSON_UNMARSHAL_ERR: %v", err)
	// 	}
	// 	resp.SetRequestParams(params)
	// 	return &resp, nil
	// }
	/* testing end */

	operation := types.GetVariations
	payload, err := params.Payload()
	if err != nil {
		return nil, err
	}
	respBody, err := c.makeRequest(ctx, operation, payload)
	if err != nil {
		return nil, err
	}

	/* Uncomment to save and read from saved item */
	// err = writeFile(fmt.Sprintf("sources/amazon/paapi5/example_response/%s_variations.json", params.ASIN), respBody)
	// if err != nil {
	// 	fmt.Println("ERROR", err)
	// }

	var resp types.GetVariationsResponse
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("JSON_UMPARSHAL_ERR: %v", err)
	}
	resp.SetRequestParams(params)
	return &resp, nil
}

// GetVariationsFromURL - get information items with variations
func (c *Client) GetVariationsFromURL(ctx context.Context, productURL string) ([]map[string]interface{}, string, error) {
	parsedURL, err := url.Parse(productURL)
	if err != nil {
		return nil, "URL_INVALID", fmt.Errorf("INVALID_URL_ERR: %v", err)
	}

	locale, err := types.NewLocale(parsedURL.Hostname())
	if err != nil {
		return nil, "DOMAIN_NOT_SUPPORTED_FOR_SOURCE", err
	}

	code, err := c.SetLocale(locale)
	if err != nil {
		return nil, code, err
	}

	asin, err := asinFromPath(parsedURL.Path)
	if err != nil {
		return nil, "ASIN_NOT_FOUND", err
	}
	params := &types.GetVariationsParams{
		ASIN:      asin,
		Condition: types.New,
		Resources: []types.Resource{
			types.BrowseNodeInfoBrowseNodes,
			types.BrowseNodeInfoBrowseNodesAncestor,
			types.BrowseNodeInfoBrowseNodesSalesRank,
			types.BrowseNodeInfoWebsiteSalesRank,
			types.ImagesPrimarySmall,
			types.ImagesPrimaryMedium,
			types.ImagesPrimaryLarge,
			types.ImagesVariantsSmall,
			types.ImagesVariantsMedium,
			types.ImagesVariantsLarge,
			types.ItemInfoByLineInfo,
			types.ItemInfoClassifications,
			types.ItemInfoContentInfo,
			types.ItemInfoContentRating,
			types.ItemInfoExternalIds,
			types.ItemInfoFeatures,
			types.ItemInfoManufactureInfo,
			types.ItemInfoProductInfo,
			types.ItemInfoTechnicalInfo,
			types.ItemInfoTitle,
			types.ItemInfoTradeInInfo,
			types.OffersListingsAvailabilityMaxOrderQuantity,
			types.OffersListingsAvailabilityMessage,
			types.OffersListingsAvailabilityMinOrderQuantity,
			types.OffersListingsAvailabilityType,
			types.OffersListingsCondition,
			types.OffersListingsConditionConditionNote,
			types.OffersListingsConditionSubCondition,
			types.OffersListingsDeliveryInfoIsAmazonFulfilled,
			types.OffersListingsDeliveryInfoIsFreeShippingEligible,
			types.OffersListingsDeliveryInfoIsPrimeEligible,
			types.OffersListingsIsBuyBoxWinner,
			types.OffersListingsLoyaltyPointsPoints,
			types.OffersListingsMerchantInfo,
			types.OffersListingsPrice,
			types.OffersListingsProgramEligibilityIsPrimeExclusive,
			types.OffersListingsProgramEligibilityIsPrimePantry,
			types.OffersListingsPromotions,
			types.OffersListingsSavingBasis,
			types.OffersSummariesHighestPrice,
			types.OffersSummariesLowestPrice,
			types.OffersSummariesOfferCount,
			types.ParentASIN,
			types.VariationSummaryPriceHighestPrice,
			types.VariationSummaryVariationDimension,
		},
		Marketplace: locale.MarketPlace(),
	}
	resp, err := c.GetVariations(ctx, params)
	if err != nil {
		return nil, "AMAZON_REQUEST_ERROR", err
	}

	// without geo_id api-core will default currency to USD
	geoID, err := locale.GeoID()
	if err != nil {
		return nil, "AMAZON_GEO_ID_ERROR", err
	}

	res, code, err := resp.Normalized(geoID)
	if err != nil {
		return nil, code, err
	}
	return res, "", nil
}
