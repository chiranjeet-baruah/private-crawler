package discovery

import (
	"log"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	htmlutils "github.com/Semantics3/sem3-go-crawl-utils/html"
	rdutils "github.com/Semantics3/sem3-go-crawl-utils/rdstore"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
)

// UpdateRdstoreData will handle rdstore data for a url
// 1. It'll add _reserved_recrawlupdate for each product
// 2. It'll omit old products from workflow.Data.Products based on existing rdstore skus
func UpdateRdstoreData(url string, isOldURL bool, workflow *types.CrawlWorkflow, appC *types.Config) *ctypes.RdstoreParentSKU {
	rdstoreData := workflow.RdstoreData

	// 1. Construct new rdstore entry for input url
	if !rdutils.CheckIfParentSKUFound(rdstoreData) {
		rdstoreData = newRdstoreEntryForURL(url, workflow)
	}

	// 2. Handle discontinued products came back live
	if htmlutils.IsSuccess(workflow.WebResponse.Status) && rdstoreData.IsDiscontinued {
		log.Printf("%s 404 product back alive, Removing the discontinued flag and updating all rdstore skus", url)
		rdstoreData.IsDiscontinued = false
		rdstoreData.DiscontinuedCounter = 0
		isOldURL = true
	}

	if isOldURL && rdstoreData.ForcediscoverSku {
		rdstoreData.ForcediscoverSku = false
	}

	// 4. Collect all the skus from crawl data and rdstore
	crawlDataSkusList := getSkusListFromCrawlData(workflow.Data.Products)
	rdstoreSkusMap := getSkusMapFromRdstore(rdstoreData)

	// 5. Make rdstore entry for all the new skus from crawl data
	for _, sku := range crawlDataSkusList {
		if _, ok := rdstoreSkusMap[sku]; !ok {
			skuData := ctypes.RdstoreChildSKU{ChildSku: sku}
			rdstoreData.Variations = append(rdstoreData.Variations, skuData)
		}
	}

	// 6. Add rdstore data to all products
	for _, product := range workflow.Data.Products {
		rdstoreData.CrawlUpdatedAt = product["time"].(int64)
		product["_reserved_recrawlupdate"] = rdstoreData
	}

	return rdstoreData
}

func newRdstoreEntryForURL(url string, workflow *types.CrawlWorkflow) (rdstoreData *ctypes.RdstoreParentSKU) {
	rdstoreData = new(ctypes.RdstoreParentSKU)
	rdstoreData.Site = workflow.DomainInfo.DomainName
	rdstoreData.ParentSku = workflow.DomainInfo.ParentSku
	rdstoreData.URL = getFinalURL(url, workflow)
	rdstoreData.Vnsp = utils.GetVnspFromWrapper(&workflow.DomainInfo.Wrapper)
	return
}

func getFinalURL(url string, workflow *types.CrawlWorkflow) (finalURL string) {
	finalURL = url
	if len(workflow.Data.Products) > 0 {
		product := workflow.Data.Products[0]
		finalURL = product["url"].(string)
	}
	return
}

func getSkusListFromCrawlData(products []map[string]interface{}) (skus []string) {
	skusMap := make(map[string]bool, 0)
	if len(products) > 0 {
		for _, product := range products {
			sku := product["sku"].(string)
			if _, ok := skusMap[sku]; !ok {
				skusMap[sku] = true
				skus = append(skus, sku)
			}
		}
	}
	return
}

func getSkusMapFromRdstore(rdstoreData *ctypes.RdstoreParentSKU) map[string]bool {
	skusMap := make(map[string]bool, 0)
	if len(rdstoreData.Variations) > 0 {
		for _, variation := range rdstoreData.Variations {
			skusMap[variation.ChildSku] = true
		}
	}
	return skusMap
}
