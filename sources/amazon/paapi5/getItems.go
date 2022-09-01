package paapi5

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Semantics3/go-crawler/sources/amazon/paapi5/types"
)

// GetItems - get information for items
func (c *Client) GetItems(ctx context.Context, params *types.GetItemsParams) (*types.GetItemsResponse, error) {
	/* Uncomment to save and read from saved item */
	// respB, err := readFile(fmt.Sprintf("sources/amazon/paapi5/example_response/%s.json", params.ItemIds[0]))
	// if err == nil {
	// 	var resp types.GetItemsResponse
	// 	err = json.Unmarshal(respB, &resp)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("JSON_UMPARSHAL_ERR: %v", err)
	// 	}
	// 	return &resp, nil
	// }
	/* testing end */

	operation := types.GetItems
	payload, err := params.Payload()
	if err != nil {
		return nil, err
	}
	respBody, err := c.makeRequest(ctx, operation, payload)
	if err != nil {
		return nil, err
	}

	/* Uncomment to save and read from saved item */
	// err = writeFile(fmt.Sprintf("sources/amazon/paapi5/example_response/%s.json", params.ItemIds[0]), respBody)
	// if err != nil {
	// 	fmt.Println("ERROR", err)
	// }

	var resp types.GetItemsResponse
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("JSON_UMPARSHAL_ERR: %v", err)
	}
	return &resp, nil
}

func (c *Client) getItemsParams(locale types.Locale, asins []string) *types.GetItemsParams {
	return &types.GetItemsParams{
		ItemIds:   asins,
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
		},
		Marketplace: locale.MarketPlace(),
	}
}

// GetItemsFromURL - get information for items
func (c *Client) GetItemsFromURL(ctx context.Context, productURL string) *types.ItemResponse {
	locale, asin, code, err := GetLocaleAsinFromURL(productURL)
	if err != nil {
		return types.NewItemResponse(nil, code, err)
	}

	code, err = c.SetLocale(locale)
	if err != nil {
		return types.NewItemResponse(nil, code, err)
	}

	params := c.getItemsParams(locale, []string{asin})
	resp, err := c.GetItems(ctx, params)
	if err != nil {
		return types.NewItemResponse(nil, "AMAZON_REQUEST_ERROR", err)
	}

	// without geo_id api-core will default currency to USD
	geoID, err := locale.GeoID()
	if err != nil {
		return types.NewItemResponse(nil, "AMAZON_GEO_ID_ERROR", err)
	}

	return resp.Normalized(geoID, []string{asin})[0]
}

// GetItemsFromAsins get multi item response
func (c *Client) GetItemsFromAsins(ctx context.Context, locale types.Locale, asins []string) []*types.ItemResponse {
	params := c.getItemsParams(locale, asins)

	code, err := c.SetLocale(locale)
	if err != nil {
		return types.NewMultiItemResponse(len(asins), code, err)
	}

	resp, err := c.GetItems(ctx, params)
	if err != nil {
		return types.NewMultiItemResponse(len(asins), "AMAZON_REQUEST_ERROR", err)
	}

	// without geo_id api-core will default currency to USD
	geoID, err := locale.GeoID()
	if err != nil {
		return types.NewMultiItemResponse(len(asins), "AMAZON_GEO_ID_ERROR", err)
	}

	res := resp.Normalized(geoID, asins)
	return res
}
