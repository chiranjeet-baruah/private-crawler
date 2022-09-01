package supervised

import (
	"fmt"
	"log"
	"time"

	"github.com/Semantics3/go-crawler/request"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
)

// Supervised type (implementing Sources interface type)
type Supervised struct {
	Name      string
	ErrorCode string
}

// GetName - return name
func (sp *Supervised) GetName() string {
	return sp.Name
}

// GetErrorCode - return code of error encountered while processing request
func (sp *Supervised) GetErrorCode() string {
	return sp.ErrorCode
}

// Request will download the webpage using proxycloud
func (sp *Supervised) Request(url string, workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (canExtract bool, code string, err error) {

	// set the error code before return (datadog metric tracking)
	defer func(c string) {
		sp.ErrorCode = code
	}(code)

	// All supervised request must have sitedetails (and wrapper) defined
	// Even for all realtime only sites, we've sitedetail & wrapper defined
	// NOTE: Following is the behaviour for DOMAIN_NOT_SUPPORTED error
	if workflow.DomainInfo.Sitedetail == nil {
		code = "DOMAIN_NOT_SUPPORTED"
		log.Printf("SUPERVISED_REQUEST: Skipping page visit for %s as domain %s doesn't have any sitedetail present\n", workflow.URL, workflow.DomainInfo.DomainName)
		return false, code, fmt.Errorf("%s doesn't have any sitedetail defined", workflow.DomainInfo.DomainName)
	}

	// 1. Construct request config and Set cache parameters
	reqConfig, code, err := pipeline.PrepareRequestConfig(workflow)
	if err != nil {
		return canExtract, code, err
	}
	cacheKey := reqConfig.CacheKey
	workflow.CacheKey = cacheKey
	workflow.CacheExpiry = pipeline.GetCacheExpiryTime()

	// 2. Read data from cache
	jobParams := workflow.JobParams
	if (jobParams != nil && jobParams.Cache == 1) || pipeline.ShouldReadFromCache(workflow) {
		utils.ReadDataFromCache(appC.ConfigData.CacheService, cacheKey, workflow)
	} else {
		workflow.CrawlTime = time.Now().Unix()
	}

	// Check if we're able download webpage from cache successfully
	if workflow.WebResponse.FromCache {
		log.Printf("SUPERVISED_REQUEST: Cache found, (%s) cache_path %s\n", workflow.URL, workflow.CacheKey)
		canExtract = true
	} else {
		log.Printf("SUPERVISED_REQUEST: Crawl start (%s)\n", url)
		workflow.WebResponse = request.VisitPage(url, &reqConfig, jobParams, &workflow.ProductMetrics, appC)
		wr := workflow.WebResponse
		utils.CollectProductMetrics("latency", wr.TimeTaken, &workflow.ProductMetrics)

		// Decides whether request is eligble for data extraction based on the response status
		// If web resp status is 500, code should be HTTP_500
		canExtract, code, err = pipeline.ValidateWebResponse(workflow)
		if err != nil {
			return canExtract, code, err
		}
	}
	return
}

func (sp *Supervised) Extract(url string, workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (code string, err error) {

	// set the error code before return (datadog metric tracking)
	defer func(c string) {
		sp.ErrorCode = code
	}(code)

	// Make rpc call to supervised extraction service
	err, errCode := extractDataUsingWrapper(url, workflow, appC)
	if err != nil {
		if workflow.Data.Status == 0 && workflow.Data.Code != "" {
			return workflow.Data.Code, err
		}
		if errCode != "" {
			errCode = "EXTRACTION_" + errCode
		} else {
			errCode = "EXTRACTION_FAILED"
		}
		code = errCode
		return errCode, err
	}

	// 5. Extract data from AJAX requests
	code, err = ExtractDataForAjaxRequests(url, workflow, appC)
	if err != nil {
		return code, err
	}

	// Handle cases of OverridingWebResponseStatus
	code, err = DefaultValidateExtractionResponse(workflow)
	if err != nil {
		return code, err
	}

	// Handle pipeline specific validations
	code, err = pipeline.ValidateExtractionResponse(workflow)
	if err != nil {
		return code, err
	}

	if workflow.FailureType != nil && *workflow.FailureType != "" {
		return *workflow.FailureType, fmt.Errorf("%s", *workflow.FailureMessage)
	}
	return
}

func (sp *Supervised) Normalize(workflow *types.CrawlWorkflow, appC *types.Config) {
	return
}
