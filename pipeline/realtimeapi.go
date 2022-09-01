package pipeline

import (
	"fmt"
	"log"
	"strings"

	"github.com/Semantics3/go-crawler/data"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	"github.com/Semantics3/sem3-go-crawl-utils/html"
	jobutils "github.com/Semantics3/sem3-go-crawl-utils/jobs"
)

type RealtimeApiPipeline struct{}

// PreCrawlOps will parse the input task to identify op and url
func (rp *RealtimeApiPipeline) PreCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (string, string, error) {
	return task, "", nil
}

// GetAllowedSiteStatusRegex - Returns allowed site status regex for supervised extraction
func (rp *RealtimeApiPipeline) GetAllowedSiteStatusRegex() string {
	return "ACTIVE|RE_SORT|INDEXING"
}

// check site status and sitedetail
// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (rp *RealtimeApiPipeline) ValidateDomainInfo(workflow *types.CrawlWorkflow) (string, error) {
	di := workflow.DomainInfo
	url := workflow.URL
	jobParams := workflow.JobParams

	sitedetail := di.Sitedetail
	siteName := di.DomainName
	isProductURL := di.IsProductUrl
	isSearchURL := di.IsSearchUrl
	extractionMode := utils.GetExtractionMode(workflow.JobParams.DataSources)

	if sitedetail != nil {
		if extractionMode == "WRAPPER" {
			if jobParams.UseSearchWrapper == 1 {
				// we also check for the search wrapper here because isSearchUrl is false by default i.e. it will be
				// false even when no searchUrlFilters are configured in the domain's sitedetail.
				if !isProductURL && sitedetail.SearchWrapperID != nil && !isSearchURL {
					return "NOT_SEARCH_PAGE", fmt.Errorf("use_search_wrapper was set and url %s is neither a product page nor a search page", url)
				} else if isSearchURL && sitedetail.SearchWrapperID == nil {
					return "DOMAININFO_SEARCH_WRAPPERID_EMPTY", fmt.Errorf("no search_wrapper_id present in sitedetail for site %s", siteName)
				}
			} else if !isProductURL {
				return "NOT_PRODUCT_PAGE", fmt.Errorf("url %s not a product page and use_search_wrapper not set", url)
			}
		} else if !isProductURL {
			// sitedetail present and urlFilters didn't match
			return "NOT_PRODUCT_PAGE", fmt.Errorf("url %s not a product page", url)
		}
	}

	// Replace workflow URL with the canonicalized version of the URL (for supervised sites) if available
	if workflow.DomainInfo.CanonicalUrl != "" {
		workflow.URL = di.CanonicalUrl
	}

	if len(workflow.JobParams.FacPools) > 0 {
		if val, ok := workflow.JobParams.FacPools[di.DomainName]; ok {
			if len(val) > 0 {
				log.Printf("Received proxy pool from FAC config. Site: %s, Val: %v\n", di.DomainName, val)
				workflow.JobParams.Pools = val
			}
		}
	}

	return "", nil
}

// Should read from rdstore: whether to read from rdstore
func (rp *RealtimeApiPipeline) ShouldReadFromRdstore(workflow *types.CrawlWorkflow) bool {
	return false
}

// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (rp *RealtimeApiPipeline) ValidateWebResponse(workflow *types.CrawlWorkflow) (bool, string, error) {
	status := workflow.WebResponse.Status
	canExtract, code, err := DefaultValidateWebResponse(workflow)
	if err != nil {
		return canExtract, code, err
	}
	// Handle permanent errors (HTTP 404)
	if html.IsPermError(status) {
		if workflow.JobParams.ExtractData == 0 {
			return false, "DOES_NOT_EXIST", fmt.Errorf("web crawl for url %s failed with a permanent error (status code: %d)", workflow.URL, status)
		}
		canExtract = true
	}
	return canExtract, "", nil
}

// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (rp *RealtimeApiPipeline) ValidateExtractionResponse(workflow *types.CrawlWorkflow) (string, error) {
	url := workflow.URL

	extractionEngine := workflow.Data.ExtractionEngine
	if (extractionEngine == "UNSUPERVISED" || extractionEngine == "DIFFBOT") && workflow.Data.Code == "NOT_PRODUCT_PAGE" {
		return "NOT_PRODUCT_PAGE", fmt.Errorf("unsupervised has detected the url %s to be not a product page", url)
	}

	if workflow.FailureType == nil {
		status := workflow.WebResponse.Status
		if html.IsPermError(status) {
			return "DOES_NOT_EXIST", fmt.Errorf("web crawl for url %s failed with a permanent error (status code: %d)", url, status)
		} else if (len(workflow.Data.Products) == 0) && !workflow.DomainInfo.IsSearchUrl {
			// Amar: Taking an opinionated decision here. If NO products could be extracted for a supervised site, it is
			// most likely because the content filters failed to apply i.e. either one or more negative content
			// filters passed or ALL content filters failed
			// In some cases, we force this to happen. For example, Amazon Parent ASIN pages
			// Even in that case, it is reasonable to fail claiming that it is NOT a `true` product page
			return "NOT_PRODUCT_PAGE", fmt.Errorf("no products could be extracted for %s", url)
		} else {
			workflow.Status = 1
		}
	}
	return "", nil
}

func (rp *RealtimeApiPipeline) PrepareRequestConfig(workflow *types.CrawlWorkflow) (types.RequestConfig, string, error) {
	reqConfig, code, err := DefaultPrepareRequestConfig(workflow)
	if err != nil {
		return reqConfig, code, err
	}
	reqConfig.CacheExpiry = rp.GetCacheExpiryTime()
	return reqConfig, code, err
}

func (rp *RealtimeApiPipeline) ShouldReadFromCache(workflow *types.CrawlWorkflow) bool {
	return false
}

// GetCacheExpiryTime - time (in seconds) before it gets expired from cache
func (rp *RealtimeApiPipeline) GetCacheExpiryTime() int32 {
	var expiry int32
	expiry = 60 * 60
	return expiry
}

func (rp *RealtimeApiPipeline) TransformError(code string, err error) (string, error) {
	if code == "HTTP_500_ERROR" {
		code = "UNREACHABLE"
	} else if code == "EXTRACTION_SITEDETAIL_EMPTY" {
		code = "DOMAIN_NOT_SUPPORTED"
	} else if code == "EXTRACTION_WRAPPER_EMPTY" {
		code = "DOMAIN_NOT_SUPPORTED"
	}
	return fmt.Sprintf("REALTIME_%s", code), err
}

func (rp *RealtimeApiPipeline) ShouldCallPostCrawlOpsOnFailure(workflow *types.CrawlWorkflow) bool {
	return false
}

// Post crawl ops for recrawl
func (rp *RealtimeApiPipeline) PostCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (code string, err error) {
	if workflow.Data.ExtractionDataSource != "WRAPPER" && workflow.Data.ExtractionDataSource != "" {
		log.Printf("SKIPPING_REALTIME_ACTIONS: (Data Source %s) not wrapper", workflow.Data.ExtractionDataSource)
		return "", nil
	}
	if strings.Contains(workflow.JobInput.JobDetails.JobType, "webhooks") {
		log.Printf("SKIPPING_REALTIME_ACTIONS: job type %s", workflow.JobInput.JobDetails.JobType)
		return "", nil
	}
	go data.RealtimeActions(workflow.URL, workflow, appC)

	// Omit all new variations during webhooks
	// which are newly obtained in crawl, but missing in skus and rdstore databases
	// Sending new variations here is causing inconsistent behaviour with
	// Vader subsequent queries on skus db through api
	jobType := jobutils.GetJobType(workflow.JobInput)
	if jobType == "webhooks_daily" || jobType == "webhooks_hourly" {
		_, oldVariations := data.GetNewOldVariations(workflow)
		workflow.Data.Products = oldVariations
	}

	// Vader needs magento sku if available to perform sale anaylytics on the number
	// of products sold directly from the website so we reuse variation_id field to
	// populate it with magento sku whenever available.
	// We are reusing variation_id as this field is alraedy used by vader and overwriting
	// variation_id field with magento sku will help directly make use of this without any changes on it's end
	// Add or change variation_id field based on product response
	// If product response contains magento sku make variation_id same as magento sku
	if jobType == "realtimeapi" && len(workflow.Data.Products) >= 1 && workflow.Data.Products[0]["magento_sku"] != nil && workflow.Data.Products[0]["magento_sku"] != "" {
		for index := range workflow.Data.Products {
			workflow.Data.Products[index]["variation_id"] = workflow.Data.Products[index]["magento_sku"]
		}
	}

	return "", nil
}
