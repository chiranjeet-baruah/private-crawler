package pipeline

import (
	"fmt"
	"log"
	"regexp"

	"github.com/Semantics3/go-crawler/data"
	"github.com/Semantics3/go-crawler/discovery"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	"github.com/Semantics3/sem3-go-crawl-utils/html"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

// DiscoveryPipeline type will have all the functions required for
// handling discovery crawl of a task
type DiscoveryPipeline struct{}

var envRegex *regexp.Regexp

// PreCrawlOps will parse the jobserver task to identify op and url
func (dp *DiscoveryPipeline) PreCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (url string, code string, err error) {
	jobInput := workflow.JobInput
	url = discovery.PrepareDiscoveryCrawlInput(task, jobInput)
	return url, "", nil
}

// ValidateDomainInfo will check site status and sitedetail
// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (dp *DiscoveryPipeline) ValidateDomainInfo(workflow *types.CrawlWorkflow) (string, error) {
	return ValidateDomainInfoForSupervised(workflow, "ACTIVE|RE_SORT|INDEXING")
}

// Should read from rdstore: whether to read from rdstore
func (dp *DiscoveryPipeline) ShouldReadFromRdstore(workflow *types.CrawlWorkflow) bool {
	return true
}

// ValidateWebResponse will return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (dp *DiscoveryPipeline) ValidateWebResponse(workflow *types.CrawlWorkflow) (bool, string, error) {
	canExtract, code, err := DefaultValidateWebResponse(workflow)
	if err != nil {
		return canExtract, code, err
	}

	url := workflow.URL
	if canExtract && utils.IsSitemapURL(url) {
		sitemapURLs, err := discovery.ExtractSitemapUrls(url, workflow.WebResponse.Content)
		if err != nil {
			err = cutils.PrintErr("SITEMAP_EXTRACTION_FAILED", fmt.Sprintf("Extracting links from sitemap %s failed with error", url), err)
			return canExtract, "", err
		}
		sitemapResponseLinks := make(map[string]ctypes.UrlMetadata, 0)
		linkMetdata := ctypes.UrlMetadata{Priority: 100}
		for _, u := range sitemapURLs {
			sitemapResponseLinks[u] = linkMetdata
		}
		workflow.Data.Links = sitemapResponseLinks
		workflow.Status = 1
		workflow.Data.Status = 1
		canExtract = false
	}

	//TODO: Cases to handle
	// 1. Product page redirecting to (active) product page
	// 		1. Perform rdstore lookup for redirect sku
	// 			1. redirect sku has NO rdstore entry
	// 			2. redirect sku has rdstore entry
	// 2. Product page redirecting to category/search page

	// if workflow.DomainInfo.IsProductUrl {
	// 	err = utils.CheckIfRedirectSkuChange(workflow)
	// 	if err != nil {
	// 		return canExtract, "REDIRECT_SKU_ERROR", err
	// 	}
	// }
	return canExtract, "", nil
}

// ValidateExtractionResponse will check extraction response (recrawl can add the EXTRACTION_FAILED_NOPRODS check here)
// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (dp *DiscoveryPipeline) ValidateExtractionResponse(workflow *types.CrawlWorkflow) (string, error) {
	url := workflow.URL
	siteName := workflow.DomainInfo.DomainName
	isProductURL := workflow.DomainInfo.IsProductUrl
	activeProds, _, totalProds := utils.GetActiveProds(url, workflow)

	if isProductURL && html.IsSuccess(workflow.WebResponse.Status) && activeProds == 0 && siteName != "amazon.com" {
		return "EXTRACTION_FAILED_NOPRODS", fmt.Errorf("CE rpc returned %d/%d active prods for successful url %s", activeProds, totalProds, url)
	}
	//TODO: Should we add any checks on 0 links extracted from category pages ?
	// Or will it create too much noise (as spidering might give us few random search / info pages)
	return "", nil
}

// PrepareRequestConfig decides if any custom configurations are to needed for proxycloud request
func (dp *DiscoveryPipeline) PrepareRequestConfig(workflow *types.CrawlWorkflow) (types.RequestConfig, string, error) {
	url := workflow.URL
	if utils.IsSitemapURL(url) {
		siteName := workflow.DomainInfo.DomainName
		reqConfig := utils.ConstructRequestConfig(url, siteName, false, workflow)
		return reqConfig, "", nil
	}

	reqConfig, code, err := DefaultPrepareRequestConfig(workflow)
	if err != nil {
		return reqConfig, code, err
	}
	if workflow.JobParams.ExtractData == 1 {
		reqConfig.CacheEvent = "on_success_or_perm_error"
	}
	reqConfig.CacheExpiry = dp.GetCacheExpiryTime()

	return reqConfig, code, err
}

// ShouldReadFromCache will check if request has to be served from cache
func (dp *DiscoveryPipeline) ShouldReadFromCache(workflow *types.CrawlWorkflow) bool {
	return false
}

// GetCacheExpiryTime - Time (in seconds) before it gets expired from cache
func (dp *DiscoveryPipeline) GetCacheExpiryTime() int32 {
	var expiry int32
	expiry = 3 * 24 * 60 * 60
	return expiry
}

// TransformError will transform the error message to known error codes for propagating upstream
func (dp *DiscoveryPipeline) TransformError(code string, err error) (string, error) {

	extractionTimeoutError := regexp.MustCompile(`failed CE rpc call: .*: RPC_TIMEOUT`)
	extractionSiteStatusError := regexp.MustCompile(`CE rpc failed for .*: Site is in .* status`)

	errorMessage := err.Error()
	if extractionTimeoutError.MatchString(errorMessage) {
		code = "EXTRACTION_RPC_TIMEOUT"
	} else if extractionSiteStatusError.MatchString(errorMessage) {
		code = "SITE_STATUS_CHECK_FAILED"
	}

	return code, err
}

func (dp *DiscoveryPipeline) ShouldCallPostCrawlOpsOnFailure(workflow *types.CrawlWorkflow) bool {
	return false
}

// PostCrawlOps will perform following actions
// 1. Adds _reserved_recrawlupdate to product data
// 2. Filter out feedback links from wrapper extracted links
// 3. Write data to rdstore
func (dp *DiscoveryPipeline) PostCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (code string, err error) {
	url := workflow.URL

	if workflow.Data.ExtractionDataSource != "WRAPPER" && workflow.Data.ExtractionDataSource != "" {
		log.Printf("SKIPPING_DISCOVERY_ACTIONS: (Data Source %s) not wrapper", workflow.Data.ExtractionDataSource)
		return "", nil
	}

	// 1. Get links to feed to jobserver queue
	discovery.FilterJobServerFeedbackLinks(url, workflow, appC)

	// 2. Skip discovery actions for staging setup
	// env := appC.ConfigData.Env
	// if envRegex == nil {
	// 	envRegex = regexp.MustCompile("production")
	// }

	// if !envRegex.MatchString(env) {
	// 	log.Printf("SKIPPING_DISCOVERY_ACTIONS: (%s) (Mode %s)\n", url, env)
	// 	return "", nil
	// }

	// 3. Perform discovery actions
	code, err = data.DiscoveryActions(url, workflow, appC)
	if err != nil {
		log.Printf("Performing discovery actions for %s failed with error %s: %v\n", url, code, err)
	}
	return
}
