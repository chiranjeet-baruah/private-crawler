package data

import (
	"log"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	rdutils "github.com/Semantics3/sem3-go-crawl-utils/rdstore"
)

// RealtimeActions will decide if product is old or new by rdstore lookup
// Old products it'll send through recrawl pipeline
func RealtimeActions(url string, workflow *types.CrawlWorkflow, appC *types.Config) (code string, err error) {
	if workflow.Status == 0 {
		log.Printf("REALTIME_ACTIONS_FAILED: Workflow failed for %s, not forwarding crawled data to recrawl/discovery pipeline\n", url)
		return
	}

	if workflow.Data.ExtractionEngine != "" && workflow.Data.ExtractionEngine != "WRAPPER" {
		log.Printf("REALTIME_ACTIONS_FAILED: Extraction engine is not WRAPPER for %s, not forwarding crawled data\n", url)
		return
	}

	if len(workflow.Data.Products) == 0 {
		log.Printf("REALTIME_ACTIONS_FAILED: No crawled products found for %s, not forwarding crawled data\n", url)
		return
	}

	isOldURL := false
	site := workflow.DomainInfo.DomainName
	redirectURL := workflow.WebResponse.Redirect

	di, err := utils.GetPartialDomainInfo(redirectURL, workflow.JobType, appC.ConfigData.WrapperServiceURI)
	if err != nil {
		log.Printf("REALTIME_ACTIONS_FAILED: INIT_URL: %s, REDIRECT_URL: %s, Fetching domain info failed with error %v\n", url, redirectURL, err)
		return "", err
	}
	if di.IsProductUrl && di.ParentSKU == "" {
		log.Printf("REALTIME_ACTIONS_FAILED: INIT_URL: %s, REDIRECT_URL: %s, Empty parent sku extracted from redirect url", url, redirectURL)
		return
	}

	// Identify if the url is already indexed into our databases
	// If so, make use of crawled data to update respective dbs
	// If a parent sku is present in rdstore
	// we can safely assume the product (and variations) is present in Skus and ES aswell
	if workflow.RdstoreData == nil {
		rdstoreData, err := rdutils.FetchParentSKU(url, site, di.ParentSKU, appC.ConfigData.RestRdstoreUpdate)
		if err != nil {
			log.Printf("RDSTORE_READ_FAILED: Performing rdstore read for SITE: %s, PARENT_SKU: %s, URL: %s failed with error: %v\n", site, di.ParentSKU, url, err)
			return "", err
		}
		if rdutils.CheckIfParentSKUFound(rdstoreData) {
			isOldURL = true
			workflow.RdstoreData = rdstoreData
		}
	}

	if isOldURL {
		log.Printf("URL: %s has been identified as old url, forwarding crawl data to recrawl pipeline", url)
		code, err = RecrawlActions(url, workflow, appC)
		if err != nil {
			log.Printf("[%s] Performing recrawl actions for %s failed with error: %v\n", code, url, err)
		}
	} else {
		// NOTE: On demand discovery is not running for old system aswell
		// code, err = DiscoveryActions(url, workflow, appC)
		// if err != nil {
		// 	err = cutils.PrintErr("DISCOVERY_ACTIONS_FAILED", fmt.Sprintf("Performing recrawl actions for %s failed with error: %s", url, code), err)
		// 	return code, err
		// }
	}

	return "", nil
}
