package unsupervised

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Semantics3/go-crawler/sources"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	jobutils "github.com/Semantics3/sem3-go-crawl-utils/jobs"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

// Unsupervised - implements Source interface and defines the respective functions
type Unsupervised struct {
	Name      string
	ErrorCode string
}

// GetName - return name
func (usp *Unsupervised) GetName() string {
	return usp.Name
}

// GetErrorCode - return code of error encountered while processing request
func (usp *Unsupervised) GetErrorCode() string {
	return usp.ErrorCode
}

// Request - Make rpc request to get the data downloaded using Unsupervised service
// Upload data to Html Cache Service
func (usp *Unsupervised) Request(url string, workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (canExtract bool, code string, err error) {

	// set the error code before return (datadog metric tracking)
	defer func(c string) {
		usp.ErrorCode = code
	}(code)

	var uResponse types.UnsupervisedResponse

	cacheKey, code, err := constructCacheKey(url, workflow)
	if err != nil {
		return canExtract, code, err
	}

	// Make Unsupervised AI request
	workflow.Data = types.ExtractionResponse{}
	requestConfig := map[string]string{"api_request_id": workflow.RequestId}
	args := []interface{}{url, requestConfig}
	err = sources.MakeRPCRequest(appC.UnsupervisedRPCClient, "UNSUPERVISED", url, "extractContent", args, &uResponse)
	if err != nil {
		if strings.Contains(err.Error(), "RPC_TIMEOUT") {
			code = "UNSUPERVISED_REQUEST_TIMEOUT"
			return canExtract, code, err
		}
		code = "UNSUPERVISED_REQUEST_FAILED"
		return canExtract, code, err
	}
	if uResponse.Status == 0 {
		err = fmt.Errorf("unsupervised response status 0, message: %s", uResponse.Message)
		code = fmt.Sprintf("UNSUPERVISED_%s", uResponse.Error) // Make use of error code returned by UCE
		return canExtract, code, err
	}
	// Write Unsupervised AI response to S3

	// This is needed as a valid cache document requires a top-level `time` field
	uResponse.Time = time.Now().Unix()
	uResponse.Content = uResponse.Html
	uResponse.Html = ""

	// Set expiry for the cache object
	var expiry int32 = 60 * 60 // seconds (1 hour)

	err = utils.WriteDataToCache(url, appC.ConfigData.CacheService, cacheKey, uResponse, expiry)
	if err != nil {
		code = "UNSUPERVISED_WRITING_TO_CACHE_FAILED"
		err = cutils.PrintErr(code, fmt.Sprintf("writing unsupervised response for %s to the cache failed\n", url), err)
		return canExtract, code, err
	} else {
		log.Printf("UNSUPERVISED: CACHE_WRITTEN: (%s) %d bytes", url, len(uResponse.Content))
		workflow.UnsupervisedCacheKey = cacheKey
		workflow.CacheKey = cacheKey
	}
	return true, "", nil
}

// Extract - Make rpc request to supervised extraction service to finish off the last mile extraction
func (usp *Unsupervised) Extract(url string, workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (code string, err error) {

	// set the error code before return (datadog metric tracking)
	defer func(c string) {
		usp.ErrorCode = code
	}(workflow.Data.Code)

	args := []interface{}{workflow}
	err = sources.MakeRPCRequest(appC.RPCClient, "UNSUPERVISED", url, "extractWithUnsupervised", args, &workflow.Data)
	if err != nil {
		workflow.Data.Status = 0
		if workflow.Data.Code == "" {
			workflow.Data.Code = "UNSUPERVISED_RPC_ERR"
		}
		return workflow.Data.Code, err
	}

	if workflow.Data.Status == 0 || workflow.Data.Code != "" {
		workflow.Data.Status = 0
		return workflow.Data.Code, errors.New(workflow.Data.Message)
	}

	// if control reaches here error is nil and workflow.data.code is equal to empty string and no
	// error is reported by UCE
	return "", nil
}

// Normalize - normalizes the data to standard schema
func (usp *Unsupervised) Normalize(workflow *types.CrawlWorkflow, appC *types.Config) {
	return
}

// Construct a cache key for an unsupervised request
func constructCacheKey(url string, workflow *types.CrawlWorkflow) (cacheKey string, code string, err error) {
	site := workflow.DomainInfo.DomainName
	jobType := jobutils.GetJobType(workflow.JobInput)
	wrapperBrowser := workflow.DomainInfo.Wrapper.Setup.Browser
	cacheId, err := utils.ConstructCacheId(url, site, jobType, workflow, wrapperBrowser)
	if err != nil {
		return "", "CONTENT_ID_GENERATION_FAILED", err
	}
	domainKey := strings.Replace(site, ".", "_", -1)
	cacheKey = fmt.Sprintf("uce/%s/%s/%s", jobType, domainKey, cacheId)
	return
}
