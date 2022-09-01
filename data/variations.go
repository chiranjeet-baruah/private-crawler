package data

import (
	"log"
	"time"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

// Separate new and old variations for a crawled web page
func GetNewOldVariations(workflow *types.CrawlWorkflow) (newVariations []map[string]interface{}, oldVariations []map[string]interface{}) {
	url := workflow.URL
	rdstoreData := workflow.RdstoreData
	crawledVariations := workflow.Data.Products

	oldVariations = make([]map[string]interface{}, 0)

	// If no rdstore data sent, all crawled variations are new
	if rdstoreData == nil || rdstoreData.URL == "" {
		log.Printf("GET_OLDNEW_RDSTORE_DATA_MISSING: (%s) No rdstore data present, so all (%d) crawled variations are new\n", url, len(crawledVariations))
		return crawledVariations, oldVariations
	}

	// Build rdstore variations lookups table for fast lookup for given URL
	rdstoreSkus := make(map[string]bool, 0)
	for _, rVariation := range rdstoreData.Variations {
		rdstoreSkus[rVariation.ChildSku] = true
	}

	// Build rdstore variations lookups table for fast lookup for given URL
	// Remove duplicate child skus in crawled variations
	crawledSkus := make(map[string]map[string]interface{}, 0)
	cv := make([]map[string]interface{}, 0)
	for _, variation := range crawledVariations {
		childSku, ok := cutils.GetStringKey(variation, "_id")
		if ok && childSku != "" {
			if _, ok = crawledSkus[childSku]; !ok {
				cv = append(cv, variation)
			}
			crawledSkus[childSku] = variation
		}
	}
	crawledVariations = cv

	// Init old & new variations lists
	nowT := time.Now().Unix()
	oldVariations = make([]map[string]interface{}, 0)
	newVariations = make([]map[string]interface{}, 0)
	geoId := utils.GetGeoIdFromWrapper(&workflow.DomainInfo.Wrapper)

	// Fetch new variations and old variations which were crawled again
	for _, variation := range crawledVariations {
		childSku, ok := cutils.GetStringKey(variation, "_id")
		if !ok {
			log.Printf("GET_OLDNEW_BAD_DATA: (%s) No _id present for child variation (%v)\n", url, variation)
			continue
		}

		_, present := rdstoreSkus[childSku]
		if !present {
			newVariations = append(newVariations, variation)
		} else {
			oldVariations = append(oldVariations, variation)
		}

		t, ok := cutils.GetInt64Key(variation, "time")
		if ok {
			nowT = t
		}
	}

	// Fetch old variations which were not crawled now
	for _, rVariation := range rdstoreData.Variations {
		childSku := rVariation.ChildSku
		variation, present := crawledSkus[childSku]
		if !present {
			variation = ConstructDiscChildSku(childSku, nowT, geoId, workflow)
			oldVariations = append(oldVariations, variation)
		}
	}

	log.Printf("GET_OLDNEW_STATUS: (%s) has %d new variations and %d old variations\n", url, len(newVariations), len(oldVariations))
	return newVariations, oldVariations
}

// Construct discontinued product data for child sku
func ConstructDiscChildSku(childSku string, nowT int64, geoId int, workflow *types.CrawlWorkflow) (discChildSkuData map[string]interface{}) {
	webResponse := workflow.WebResponse
	discChildSkuData = map[string]interface{}{
		"_id":                childSku,
		"crawl_id":           childSku,
		"sku":                childSku,
		"url":                webResponse.Redirect,
		"_reserved_init_url": webResponse.URL,
		"time":               nowT,
		"isdiscontinued":     "1",
		"recentoffers_count": 0,
	}
	if geoId != 0 {
		discChildSkuData["geo_id"] = geoId
	}
	return discChildSkuData
}
