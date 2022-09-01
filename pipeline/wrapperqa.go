package pipeline

import (
	"regexp"

	"github.com/Semantics3/go-crawler/types"
)

type WrapperQAPipeline struct{}

// PreCrawlOps will parse the jobserver task to identify op and url
func (wp *WrapperQAPipeline) PreCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (string, string, error) {
	return task, "", nil
}

// check site status and sitedetail
// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (wp *WrapperQAPipeline) ValidateDomainInfo(workflow *types.CrawlWorkflow) (string, error) {
	return ValidateDomainInfoForSupervised(workflow, "ACTIVE|RE_SORT|PAUSE|RECRAWL")
}

// check web response (recrawl can add the REDIRECT_SKU_ERROR check here)
// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (wp *WrapperQAPipeline) ValidateWebResponse(workflow *types.CrawlWorkflow) (bool, string, error) {
	canExtract, code, err := DefaultValidateWebResponse(workflow)
	if err != nil {
		return canExtract, code, err
	}

	return canExtract, "", nil
}

// check extraction response (recrawl can add the EXTRACTION_FAILED_NOPRODS check here)
// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (wp *WrapperQAPipeline) ValidateExtractionResponse(workflow *types.CrawlWorkflow) (string, error) {
	return "", nil
}

func (wp *WrapperQAPipeline) PrepareRequestConfig(workflow *types.CrawlWorkflow) (types.RequestConfig, string, error) {
	reqConfig, code, err := DefaultPrepareRequestConfig(workflow)
	if err != nil {
		return reqConfig, code, err
	}
	reqConfig.CacheExpiry = wp.GetCacheExpiryTime()
	return reqConfig, code, err
}

func (wp *WrapperQAPipeline) ShouldReadFromCache(workflow *types.CrawlWorkflow) bool {
	if workflow.JobParams.Cache == 1 {
		return true
	}
	return false
}

// GetCacheExpiryTime - time (in seconds) before it gets expired from cache
func (rp *WrapperQAPipeline) GetCacheExpiryTime() int32 {
	var expiry int32
	expiry = 12 * 60 * 60
	return expiry
}

func (wp *WrapperQAPipeline) TransformError(code string, err error) (string, error) {
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

func (rp *WrapperQAPipeline) ShouldCallPostCrawlOpsOnFailure(workflow *types.CrawlWorkflow) bool {
	return false
}

// Should read from rdstore: whether to read from rdstore
func (sp *WrapperQAPipeline) ShouldReadFromRdstore(workflow *types.CrawlWorkflow) bool {
	return true
}

// Post crawl ops for wrapperqa
func (wp *WrapperQAPipeline) PostCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (code string, err error) {
	return "", nil
}
