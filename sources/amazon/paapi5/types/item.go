package types

import (
	"fmt"
	"net/url"
)

// Item represents Item object in json response
type Item struct {
	ASIN                string               `json:"ASIN,omitempty"`
	BrowseNodeInfo      BrowseNodeInfo       `json:"BrowseNodeInfo,omitempty"`
	DetailPageURL       string               `json:"DetailPageURL,omitempty"`
	Images              Images               `json:"Images,omitempty"`
	ItemInfo            ItemInfo             `json:"ItemInfo,omitempty"`
	Offers              Offers               `json:"Offers,omitempty"`
	ParentASIN          string               `json:"ParentASIN,omitempty"`
	RentalOffers        RentalOffers         `json:"RentalOffers,omitempty"`
	Score               float32              `json:"Score,omitempty"`
	VariationAttributes []VariationAttribute `json:"VariationAttributes,omitempty"`
}

// Normalize normalizes data to sem3 specific format
func (i *Item) Normalize(geoID int) map[string]interface{} {
	res := make(map[string]interface{})
	/**********	Item Info ******/
	res["name"] = i.ItemInfo.Title.DisplayValue
	res["sku"] = i.ASIN
	res["geo_id"] = geoID
	features := map[string]interface{}{}
	features["ASIN"] = i.ASIN

	amzURL, err := url.Parse(i.DetailPageURL)
	if err == nil {
		res["url"] = fmt.Sprintf("%s://%s%s", amzURL.Scheme, amzURL.Host, amzURL.Path)
	}

	if len(i.ItemInfo.ExternalIds.EANs.DisplayValues) > 0 {
		res["ean"] = i.ItemInfo.ExternalIds.EANs.DisplayValues
	}
	if len(i.ItemInfo.ExternalIds.UPCs.DisplayValues) > 0 {
		res["upc"] = i.ItemInfo.ExternalIds.EANs.DisplayValues
	}
	if i.ItemInfo.ByLineInfo.Brand.DisplayValue != "" {
		res["brand"] = i.ItemInfo.ByLineInfo.Brand.DisplayValue
		features["Brand"] = i.ItemInfo.ByLineInfo.Brand.DisplayValue
	}
	if i.ItemInfo.ByLineInfo.Manufacturer.DisplayValue != "" {
		res["manufacturer"] = i.ItemInfo.ByLineInfo.Manufacturer.DisplayValue
		features["Manufacturer"] = i.ItemInfo.ByLineInfo.Manufacturer.DisplayValue
	}
	if i.ItemInfo.ManufactureInfo.Model.DisplayValue != "" {
		res["model"] = i.ItemInfo.ManufactureInfo.Model.DisplayValue
		features["Item model number"] = i.ItemInfo.ManufactureInfo.Model.DisplayValue
	}

	if len(i.ItemInfo.Features.DisplayValues) > 0 {
		features["blob"] = i.ItemInfo.Features.DisplayValues
	}

	// Variation Id info
	variations_attribute_length := i.VariationAttributes
	if len(variations_attribute_length) >= 1 {
		if i.ParentASIN != "" {
			res["variation_id"] = i.ParentASIN
		}
	}

	// ProductInfo
	productInfo := i.ItemInfo.ProductInfo
	if productInfo.Color.DisplayValue != "" {
		res["color"] = productInfo.Color.DisplayValue
	}
	itemDimensions := productInfo.ItemDimensions
	if itemDimensions.Height.DisplayValue != 0 {
		res["height"] = fmt.Sprintf("%v", itemDimensions.Height.DisplayValue)
		res["height_unit"] = itemDimensions.Height.Unit
	}
	if itemDimensions.Length.DisplayValue != 0 {
		res["length"] = fmt.Sprintf("%v", itemDimensions.Length.DisplayValue)
		res["length_unit"] = itemDimensions.Length.Unit
	}
	if itemDimensions.Width.DisplayValue != 0 {
		res["width"] = fmt.Sprintf("%v", itemDimensions.Width.DisplayValue)
		res["width_unit"] = itemDimensions.Width.Unit
	}
	if itemDimensions.Weight.DisplayValue != 0 {
		res["weight"] = fmt.Sprintf("%v", itemDimensions.Weight.DisplayValue)
		res["weight_unit"] = itemDimensions.Weight.Unit
		features["Item Weight"] = fmt.Sprintf("%v %s", res["weight"], res["weight_unit"])
	}
	if dimensions := itemDimensions.GetDimension(); dimensions != "" {
		res["dimensions"] = dimensions
		features["Item Dimensions LxWxH"] = dimensions
	}
	if unitCount := productInfo.UnitCount.DisplayValue; unitCount != 0 {
		res["package_quantity"] = unitCount
	}

	// images
	res["images"] = i.Images.GetImages()

	/*********** Offers ******/
	// jStr, _ := json.MarshalIndent(i, "", "  ")
	// fmt.Printf("OFFERS DATA: %v\n", string(jStr))

	offers := i.Offers.GetOffers()
	res["offers"] = offers

	if len(offers) > 0 {
		res["listprice"] = offers[0]["price"]
		res["listprice_currency"] = offers[0]["currency"]
		if i.Offers.Listings[0].SavingBasis.Amount != 0 {
			res["listprice"] = fmt.Sprintf("%v", i.Offers.Listings[0].SavingBasis.Amount)
		}
	}

	offerSummary := i.Offers.Summaries

	for _, summ := range offerSummary {
		if summ.Condition.Value == "New" {
			res["offers_count"] = summ.OfferCount
		}
	}

	/*********** Category ******/
	browserNodeExtract := i.BrowseNodeInfo.ExtractInfo()
	if browserNodeExtract != nil {
		res["crumb"] = browserNodeExtract.Crumb
		if browserNodeExtract.SalesRankCategory != "" {
			res["salesrank_category"] = browserNodeExtract.SalesRankCategory
		}
		if browserNodeExtract.SalesRank != 0 {
			res["salesrank"] = fmt.Sprintf("%d", browserNodeExtract.SalesRank)
		}
		if browserNodeExtract.CategoryPath != "" {
			res["categorypath"] = browserNodeExtract.CategoryPath
		}
		if browserNodeExtract.CategoryRank != 0 {
			res["categoryrank"] = fmt.Sprintf("%d", browserNodeExtract.CategoryRank)
		}
	}

	if len(features) > 0 {
		res["features"] = features
	}
	return res
}
