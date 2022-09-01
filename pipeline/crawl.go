package pipeline

import (
	"fmt"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	"github.com/Semantics3/sem3-go-crawl-utils/html"
)

// CrawlPipeline is an interface which holds all the ondemand crawl pipeline specific functions
type CrawlPipeline struct{}

// PreCrawlOps will parse the jobserver task to identify op and url
func (cp *CrawlPipeline) PreCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (string, string, error) {
	return task, "", nil
}

// ValidateDomainInfo will verify if domainInfo is valid based on type of request
func (cp *CrawlPipeline) ValidateDomainInfo(workflow *types.CrawlWorkflow) (string, error) {
	return ValidateDomainInfoForSupervised(workflow, "ACTIVE|RE_SORT|INDEXING")
}

// ShouldReadFromRdstore whether to read from rdstore
func (cp *CrawlPipeline) ShouldReadFromRdstore(workflow *types.CrawlWorkflow) bool {
	return false
}

// ValidateWebResponse will validate web response and decides whether request is sucessful or not
func (rp *CrawlPipeline) ValidateWebResponse(workflow *types.CrawlWorkflow) (bool, string, error) {
	canExtract, code, err := DefaultValidateWebResponse(workflow)
	if err != nil {
		return canExtract, code, err
	}
	return canExtract, "", nil
}

// ValidateExtractionResponse will verify the response received from extraction service
func (cp *CrawlPipeline) ValidateExtractionResponse(workflow *types.CrawlWorkflow) (string, error) {
	url := workflow.URL
	siteName := workflow.DomainInfo.DomainName
	isProductUrl := workflow.DomainInfo.IsProductUrl

	activeProds, _, totalProds := utils.GetActiveProds(url, workflow)
	if isProductUrl && html.IsSuccess(workflow.WebResponse.Status) && activeProds == 0 && siteName != "amazon.com" {
		return "EXTRACTION_FAILED_NOPRODS", fmt.Errorf("CE rpc returned %d/%d active prods for successful url %s", activeProds, totalProds, url)
	}

	return "", nil
}

// PrepareRequestConfig will construct pipeline specific web request
func (cp *CrawlPipeline) PrepareRequestConfig(workflow *types.CrawlWorkflow) (types.RequestConfig, string, error) {
	reqConfig, code, err := DefaultPrepareRequestConfig(workflow)
	if err != nil {
		return reqConfig, code, err
	}
	reqConfig.CacheExpiry = cp.GetCacheExpiryTime()
	return reqConfig, code, err
}

// ShouldReadFromCache decides whether to download webpage from website or read it from cache
func (cp *CrawlPipeline) ShouldReadFromCache(workflow *types.CrawlWorkflow) bool {
	return false
}

// GetCacheExpiryTime - Time (in seconds) before it gets expired from cache
func (cp *CrawlPipeline) GetCacheExpiryTime() int32 {
	var expiry int32
	expiry = 1 * 24 * 60 * 60
	return expiry
}

// TransformError will convert the internal error codes to proper jobserver error codes
func (cp *CrawlPipeline) TransformError(code string, err error) (string, error) {
	return code, err
}

// ShouldCallPostCrawlOpsOnFailure will decide if post crawl ops on failures at any level
func (cp *CrawlPipeline) ShouldCallPostCrawlOpsOnFailure(workflow *types.CrawlWorkflow) bool {
	return false
}

// PostCrawlOps will perform actions needed after crawling/extracting
func (cp *CrawlPipeline) PostCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (code string, err error) {
	return "", nil
}
