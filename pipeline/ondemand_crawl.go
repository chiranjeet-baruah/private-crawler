package pipeline

import (
	"fmt"
	"log"
	"regexp"
	"strconv"

	"github.com/Semantics3/go-crawler/data"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	"github.com/Semantics3/sem3-go-crawl-utils/html"
)

// OnDemandCrawlPipeline is an interface which holds all the ondemand crawl pipeline specific functions
type OnDemandCrawlPipeline struct{}

// PreCrawlOps will parse the jobserver task to identify op and url
func (cp *OnDemandCrawlPipeline) PreCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (string, string, error) {
	// Parse input task
	var url string
	matches, didMatch, _ := utils.FindStringSubmatch(task, `^ln\_(\d+)\;(.*)`, "")
	if didMatch {
		_, err := strconv.Atoi(matches[1])
		if err != nil {
			return "", "BAD_INPUT", err
		}
		url = matches[2]
	} else {
		// Handle the cases where proper urls are sent as tasks
		if utils.IsURL(task) {
			url = task
		}
	}

	// Get retry count and update pools accordingly
	// workflow.JobParams.Pools = getProxyPoolForCurrentAttempt(task, workflow)
	return url, "", nil
}

// ValidateDomainInfo will verify if domainInfo is valid based on type of request
func (cp *OnDemandCrawlPipeline) ValidateDomainInfo(workflow *types.CrawlWorkflow) (string, error) {
	return ValidateDomainInfoForSupervised(workflow, "ACTIVE|RE_SORT|INDEXING")
}

// ShouldReadFromRdstore whether to read from rdstore
func (cp *OnDemandCrawlPipeline) ShouldReadFromRdstore(workflow *types.CrawlWorkflow) bool {
	return false
}

// ValidateWebResponse will validate web response and decides whether request is sucessful or not
func (cp *OnDemandCrawlPipeline) ValidateWebResponse(workflow *types.CrawlWorkflow) (bool, string, error) {
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

// ValidateExtractionResponse will verify the response received from extraction service
func (cp *OnDemandCrawlPipeline) ValidateExtractionResponse(workflow *types.CrawlWorkflow) (string, error) {
	return "", nil
}

// PrepareRequestConfig will construct pipeline specific web request
func (cp *OnDemandCrawlPipeline) PrepareRequestConfig(workflow *types.CrawlWorkflow) (types.RequestConfig, string, error) {
	reqConfig, code, err := DefaultPrepareRequestConfig(workflow)
	if err != nil {
		return reqConfig, code, err
	}
	reqConfig.CacheExpiry = cp.GetCacheExpiryTime()
	return reqConfig, code, err
}

// ShouldReadFromCache decides whether to download webpage from website or read it from cache
func (cp *OnDemandCrawlPipeline) ShouldReadFromCache(workflow *types.CrawlWorkflow) bool {
	return false
}

// GetCacheExpiryTime - time (in seconds) before it gets expired from cache
func (cp *OnDemandCrawlPipeline) GetCacheExpiryTime() int32 {
	var expiry int32
	expiry = 60 * 60
	return expiry
}

// TransformError will convert the internal error codes to proper jobserver error codes
func (cp *OnDemandCrawlPipeline) TransformError(code string, err error) (string, error) {
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

// ShouldCallPostCrawlOpsOnFailure will decide if post crawl ops on failures at any level
func (cp *OnDemandCrawlPipeline) ShouldCallPostCrawlOpsOnFailure(workflow *types.CrawlWorkflow) bool {
	return true
}

// PostCrawlOps will perform actions needed after crawling/extracting
func (cp *OnDemandCrawlPipeline) PostCrawlOps(task string, workflow *types.CrawlWorkflow, appC *types.Config) (code string, err error) {
	if workflow.Data.ExtractionDataSource != "WRAPPER" && workflow.Data.ExtractionDataSource != "" {
		log.Printf("SKIPPING_ONDEMAND_ACTIONS: (Data Source %s) not wrapper", workflow.Data.ExtractionDataSource)
		return "", nil
	}
	// Handle failed tasks due to temp errors
	// if html.IsTempError(workflow.WebResponse.Status) {
	// 	retryCount := workflow.JobInput.Tasks[task].STRetryCount
	// 	if retryCount < 10 {
	// 		metadata := ctypes.UrlMetadata{
	// 			STRetryCount: retryCount + 1,
	// 		}
	// 		workflow.Data.Links = make(map[string]ctypes.UrlMetadata)
	// 		workflow.Data.Links[task] = metadata
	// 		workflow.SendFailureAsFeedback = true
	// 		log.Println("Retrying failed task through jobserver for ", task)
	// 	} else {
	// 		log.Printf("TASK: %s, RETRY_COUNT: %d, Not performing retry for the task as it exceeded max retries", task, retryCount)
	// 	}
	// }
	go data.OnDemandCrawlActions(task, workflow, appC)
	return "", nil
}

func getProxyPoolForCurrentAttempt(task string, workflow *types.CrawlWorkflow) (pool []string) {
	// Get current attempt
	var attempt int
	for t, metadata := range workflow.JobInput.Tasks {
		if t == task {
			attempt = metadata.STRetryCount + 1
		}
	}

	// Hardcoded proxy pools
	pool1 := []string{"internal", "proxybonanza_us_1_exclusive"}
	pool2 := []string{"internal_realtime", "proxybonanza_us_2_exclusive"}
	pool3 := []string{"internal_chrome", "proxybonanza_us_3_exclusive"}
	pool4 := []string{"internal_firefox", "proxybonanza_us_4_exclusive"}
	pool5 := []string{"gcp::internal", "proxybonanza_us_5_exclusive"}
	poolMap := map[int][]string{
		1:  pool1,
		2:  pool1,
		3:  pool2,
		4:  pool2,
		5:  pool3,
		6:  pool3,
		7:  pool4,
		8:  pool4,
		9:  pool5,
		10: pool5,
	}

	pool = poolMap[attempt]
	log.Printf("TASK: %s, CURRENT_ATTEMPT: %d, PROXY_POOL: %v\n", task, attempt, pool)
	return
}
