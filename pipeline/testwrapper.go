package pipeline

import (
	"fmt"
	"log"
	"regexp"

	"github.com/Semantics3/go-crawler/data"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	"github.com/Semantics3/sem3-go-crawl-utils/html"
	htmlutils "github.com/Semantics3/sem3-go-crawl-utils/html"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	validatelib "github.com/Semantics3/sem3-go-crawl-utils/validate"
)

type TestWrapperPipeline struct{}

// PreCrawlOps will parse the jobserver task to identify op and url
func (rp *TestWrapperPipeline) PreCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (string, string, error) {
	return task, "", nil
}

// check site status and sitedetail
// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
// Site can be in any status and might not have a status also (for a new wrapper)
func (rp *TestWrapperPipeline) ValidateDomainInfo(workflow *types.CrawlWorkflow) (string, error) {
	return ValidateDomainInfoForSupervised(workflow, "\\w+|")
}

// Should read from rdstore: whether to read from rdstore
func (tp *TestWrapperPipeline) ShouldReadFromRdstore(workflow *types.CrawlWorkflow) bool {
	if workflow.JobInput.DataPipeline != nil && workflow.JobInput.DataPipeline.AsRecrawl {
		return true
	} else {
		return false
	}
}

// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (rp *TestWrapperPipeline) ValidateWebResponse(workflow *types.CrawlWorkflow) (bool, string, error) {
	canExtract, code, err := DefaultValidateWebResponse(workflow)
	if err != nil {
		return canExtract, code, err
	}
	return canExtract, "", nil
}

// check extraction response (recrawl can add the EXTRACTION_FAILED_NOPRODS check here)
// return an error in case of validation failure and nil otherwise. nil implies that the
// workflow can continue
func (rp *TestWrapperPipeline) ValidateExtractionResponse(workflow *types.CrawlWorkflow) (string, error) {
	url := workflow.URL
	siteName := workflow.DomainInfo.DomainName
	isProductUrl := workflow.DomainInfo.IsProductUrl

	activeProds, _, totalProds := utils.GetActiveProds(url, workflow)
	if isProductUrl && html.IsSuccess(workflow.WebResponse.Status) && activeProds == 0 && siteName != "amazon.com" {
		return "EXTRACTION_FAILED_NOPRODS", fmt.Errorf("CE rpc returned %d/%d active prods for successful url %s", activeProds, totalProds, url)
	}

	return "", nil
}

func (rp *TestWrapperPipeline) PrepareRequestConfig(workflow *types.CrawlWorkflow) (types.RequestConfig, string, error) {
	reqConfig, code, err := DefaultPrepareRequestConfig(workflow)
	if err != nil {
		return reqConfig, code, err
	}
	reqConfig.CacheExpiry = rp.GetCacheExpiryTime()
	return reqConfig, code, err
}

func (rp *TestWrapperPipeline) ShouldReadFromCache(workflow *types.CrawlWorkflow) bool {
	return false
}

// GetCacheExpiryTime - time (in seconds) before it gets expired from cache
func (rp *TestWrapperPipeline) GetCacheExpiryTime() int32 {
	var expiry int32
	expiry = 24 * 60 * 60
	return expiry
}

func (rp *TestWrapperPipeline) TransformError(code string, err error) (string, error) {
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

func (tp *TestWrapperPipeline) ShouldCallPostCrawlOpsOnFailure(workflow *types.CrawlWorkflow) bool {
	return false
}

// PostCrawlOps for testwrapper
func (rp *TestWrapperPipeline) PostCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (code string, err error) {
	if workflow.Data.ExtractionDataSource != "WRAPPER" && workflow.Data.ExtractionDataSource != "" {
		log.Printf("SKIPPING_TESTWRAPPER_ACTIONS: (Data Source %s) not wrapper", workflow.Data.ExtractionDataSource)
		return "", nil
	}

	overridingHttpStatus := workflow.Data.OverridingWebResponseStatus
	log.Printf("S3_CACHE_READ_POST: (%s, ttl %d), overridingHttpStatus %d\n", workflow.CacheKey, workflow.JobParams.CacheTtl, overridingHttpStatus)

	utils.ReadDataFromCache(appC.ConfigData.CacheService, workflow.CacheKey, workflow)

	// Take the CE response after updating worflow.webResponse from cache service
	if overridingHttpStatus > 0 {
		workflow.WebResponse.Status = overridingHttpStatus
		workflow.WebResponse.Success = htmlutils.IsSuccess(workflow.WebResponse.Status)
	}

	// If as_recrawl is sent in, mark SKUS from rdstore as discontinued
	if workflow.JobInput.DataPipeline != nil && workflow.JobInput.DataPipeline.AsRecrawl {
		data.PrepareDataForRecrawlETL(workflow)
		if workflow.Status == 0 && workflow.FailureType != nil && *workflow.FailureType != "" {
			return *workflow.FailureType, fmt.Errorf("%v", *workflow.FailureMessage)
		}
	}

	// Validate data
	siteName := workflow.DomainInfo.DomainName
	url := workflow.URL
	if workflow.Data.Products != nil && len(workflow.Data.Products) > 0 {
		workflow.ValidateErrors = &ctypes.ValidateErrs{
			Errs: make([]string, 0),
			Warn: make([]string, 0),
		}
		for _, productData := range workflow.Data.Products {
			errs, warn, err := validatelib.ValidateRawData(siteName, url, productData, "SKUS")
			if err != nil {
				return "VALIDATE_DATA_FAIL", fmt.Errorf("failed to validate data: %v", err)
			}
			if errs != nil && len(errs) > 0 {
				workflow.ValidateErrors.Errs = append(workflow.ValidateErrors.Errs, errs...)
			}
			if warn != nil && len(warn) > 0 {
				workflow.ValidateErrors.Warn = append(workflow.ValidateErrors.Warn, warn...)
			}
		}
		workflow.ValidateErrors.Errs = TransformArrStr(validatelib.DedupErrsByField(workflow.ValidateErrors.Errs))
		workflow.ValidateErrors.Warn = TransformArrStr(validatelib.DedupErrsByField(workflow.ValidateErrors.Warn))
	}
	return "", nil
}

var r1 *regexp.Regexp = regexp.MustCompile(`^\[[^\]]*\]`)
var r2 *regexp.Regexp = regexp.MustCompile(`\s+\([^\)]*\)$`)

// Transform arr str
func TransformArrStr(errs []string) []string {
	if errs == nil || len(errs) == 0 {
		return errs
	}
	f := make([]string, 0)
	for _, errMsg := range errs {
		errMsg = r1.ReplaceAllString(errMsg, "")
		errMsg = r2.ReplaceAllString(errMsg, "")
		f = append(f, errMsg)
	}
	return f
}
