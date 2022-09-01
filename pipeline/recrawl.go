package pipeline

import (
	"fmt"
	"log"
	"regexp"

	"github.com/Semantics3/go-crawler/data"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	"github.com/Semantics3/sem3-go-crawl-utils/html"
	// jobutils "github.com/Semantics3/sem3-go-crawl-utils/jobs"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
)

type RecrawlPipeline struct{}

// PreCrawlOps will parse the jobserver task to identify op and url
func (rp *RecrawlPipeline) PreCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (string, string, error) {
	return task, "", nil
}

// check site status and sitedetail
// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (rp *RecrawlPipeline) ValidateDomainInfo(workflow *types.CrawlWorkflow) (string, error) {
	return ValidateDomainInfoForSupervised(workflow, "ACTIVE|RE_SORT")
}

// Should read from rdstore: whether to read from rdstore
func (rp *RecrawlPipeline) ShouldReadFromRdstore(workflow *types.CrawlWorkflow) bool {
	return true
}

// check web response (recrawl can add the REDIRECT_SKU_ERROR check here)
// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (rp *RecrawlPipeline) ValidateWebResponse(workflow *types.CrawlWorkflow) (bool, string, error) {
	canExtract, code, err := DefaultValidateWebResponse(workflow)
	if err != nil {
		return canExtract, code, err
	}

	return canExtract, "", nil
}

// check extraction response (recrawl can add the EXTRACTION_FAILED_NOPRODS check here)
// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
// REDIRECT_SKU_ERROR has been moved here to identify robot block pages
func (rp *RecrawlPipeline) ValidateExtractionResponse(workflow *types.CrawlWorkflow) (string, error) {
	url := workflow.URL
	siteName := workflow.DomainInfo.DomainName
	isProductUrl := workflow.DomainInfo.IsProductUrl

	if html.IsSuccess(workflow.WebResponse.Status) {
		err := utils.CheckIfRedirectSkuChange(workflow)
		if err != nil {
			// Empty products array
			workflow.Data.Products = make([]map[string]interface{}, 0)
			return "REDIRECT_SKU_ERROR", err
		}

		activeProds, _, totalProds := utils.GetActiveProds(url, workflow)
		if isProductUrl && activeProds == 0 && siteName != "amazon.com" {
			return "EXTRACTION_FAILED_NOPRODS", fmt.Errorf("CE rpc returned %d/%d active prods for successful url %s", activeProds, totalProds, url)
		}
	}

	return "", nil
}

func (rp *RecrawlPipeline) PrepareRequestConfig(workflow *types.CrawlWorkflow) (types.RequestConfig, string, error) {
	reqConfig, code, err := DefaultPrepareRequestConfig(workflow)
	if err != nil {
		return reqConfig, code, err
	}
	reqConfig.CacheExpiry = rp.GetCacheExpiryTime()
	return reqConfig, code, err
}

func (rp *RecrawlPipeline) ShouldReadFromCache(workflow *types.CrawlWorkflow) bool {
	return false
}

// GetCacheExpiryTime - time (in seconds) before it gets expired from cache
func (rp *RecrawlPipeline) GetCacheExpiryTime() int32 {
	var expiry int32
	expiry = 60 * 60
	return expiry
}

func (rp *RecrawlPipeline) TransformError(code string, err error) (string, error) {
	rdstoreTimeout := regexp.MustCompile(`Client.Timeout exceeded while awaiting headers`)
	extractionTimeoutError := regexp.MustCompile(`failed CE rpc call: .*: RPC_TIMEOUT`)
	extractionSiteStatusError := regexp.MustCompile(`CE rpc failed for .*: Site is in .* status`)

	errorMessage := err.Error()
	if extractionTimeoutError.MatchString(errorMessage) {
		code = "EXTRACTION_RPC_TIMEOUT"
	} else if extractionSiteStatusError.MatchString(errorMessage) {
		code = "SITE_STATUS_CHECK_FAILED"
	} else if code == "RDSTORE_READ_FAIL" && rdstoreTimeout.MatchString(errorMessage) {
		code = "RDSTORE_READ_TIMEOUT"
	} else if code == "RDSTORE_WRITE_FAILED" && rdstoreTimeout.MatchString(errorMessage) {
		code = "RDSTORE_WRITE_TIMEOUT"
	}
	return code, err
}

func (rp *RecrawlPipeline) ShouldCallPostCrawlOpsOnFailure(workflow *types.CrawlWorkflow) bool {
	return false
}

// Post crawl ops for recrawl
func (rp *RecrawlPipeline) PostCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (code string, err error) {
	url := workflow.URL
	if workflow.Data.ExtractionDataSource != "WRAPPER" && workflow.Data.ExtractionDataSource != "" {
		log.Printf("SKIPPING_RECRAWL_ACTIONS: (Data Source %s) not wrapper", workflow.Data.ExtractionDataSource)
		return "", nil
	}

	// This error check is now performed by pipeline/executor.go
	// if workflow.Status == 0 && workflow.FailureType != nil && *workflow.FailureType != "" {
	// 	return *workflow.FailureType, fmt.Errorf("%v", *workflow.FailureMessage)
	// }

	// Skip recrawl actions if not prod mode or upon http 5xx failures
	// env := appC.ConfigData.Env
	// httpStatus := workflow.WebResponse.Status
	// jobType := jobutils.GetJobType(workflow.JobInput)
	// if env == "dev" || env == "staging" {
	// 	log.Printf("SKIPPING_RECRAWL_ACTIONS: (%s) (Mode %s, HTTP Status %d, jobType : %s)\n", url, env, httpStatus, jobType)
	// 	return "", nil
	// }

	// Below this point are actions: Updating rdstore and pushing messages to ETL
	code, err = data.RecrawlActions(url, workflow, appC)

	// Recrawl should not send links as feedback to jobserver
	if len(workflow.Data.Links) > 0 {
		log.Printf("%d links extracted for %s, Emptying links for recrawl job_type", len(workflow.Data.Links), workflow.URL)
		workflow.Data.Links = make(map[string]ctypes.UrlMetadata, 0)
	}
	return
}
