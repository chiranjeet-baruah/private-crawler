package pipeline

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/Semantics3/go-crawler/data"
	"github.com/Semantics3/go-crawler/merge"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	"github.com/Semantics3/sem3-go-crawl-utils/html"
	jobutils "github.com/Semantics3/sem3-go-crawl-utils/jobs"
	rdutils "github.com/Semantics3/sem3-go-crawl-utils/rdstore"
	rh "github.com/Semantics3/sem3-go-crawl-utils/redis"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	"github.com/gomodule/redigo/redis"
)

// Crawl pipeline executor
// Executes webcrawl lifecycle of a URL for diff jobtypes (recrawl|discovery_crawl|realtime)
func PipelineExecutor(task string, jobInput *ctypes.Batch, pipeline types.Pipeline, appC *types.Config, queueName string) (workflow *types.CrawlWorkflow) {

	var url string
	var code string
	var err error

	// 1. Init workflow
	start := time.Now()
	workflow = &types.CrawlWorkflow{
		URL:       url,
		JobInput:  jobInput,
		QueueName: queueName,
	}

	// 2. Decode crawl job params
	var jobParams *ctypes.CrawlJobParams
	jobParams, err = utils.ParseJobParams(url, workflow.JobInput.JobParams)
	if err != nil {
		utils.FailWorkflow(task, pipeline, workflow, "JOBPARAMS_READERR", err.Error(), appC)
		return workflow
	}
	workflow.JobParams = jobParams

	// 3. Parse input task to get url
	url, code, err = pipeline.PreCrawlOps(task, workflow, appC)
	workflow.URL = url
	if err != nil {
		workflow.PreCrawlOpsFailed = true
		utils.FailWorkflow(task, pipeline, workflow, code, err.Error(), appC)
		return workflow
	}

	// 4. Retrieve domain info from wrapper-service
	workflow.JobType = jobutils.GetJobType(jobInput)
	workflow.DomainInfo, err = utils.GetCompleteDomainInfo(workflow.URL, workflow.JobType, appC.ConfigData.WrapperServiceURI, workflow.JobParams)
	if err != nil {
		utils.FailWorkflow(task, pipeline, workflow, "RETRIEVE_DOMAIN_INFO_FAIL", err.Error(), appC)
		return workflow
	}

	// 5. Add default values

	// NOTE: Handle the cases where both data_sources & merge_mode keys are missing in input
	// None of recrawl/discovery/testwrapper/wrapperqa pipelines will send those params
	// We should assign wrapper as default ourselves
	handleMissingKeysFromInput(workflow, appC)

	// 6. Validate domain info
	code, err = pipeline.ValidateDomainInfo(workflow)
	if err != nil {
		utils.FailWorkflow(task, pipeline, workflow, code, err.Error(), appC)
		return workflow
	}

	// 7. Initiate object for product metrics
	// Product metrics is where we track all url level metrics
	// What's the time taken for domain/info network call
	// What's the latency (across all network calls made incl extraction)
	// What's the total extraction time etc
	siteName := workflow.DomainInfo.DomainName
	parentSku := workflow.DomainInfo.ParentSku
	workflow.ProductMetrics = types.ProductMetrics{
		Site:             siteName,
		JobType:          workflow.JobType,
		Customer:         jobParams.Customer,
		RecrawlFrequency: jobParams.RecrawlFrequency,
		Extraction:       0.0,
		Total:            0.0,
		Latency:          0.0,
		DomainInfo:       0.0,
		UrlCount:         0,
		RetryCount:       0,
		Value:            0,
	}

	// 8. Assign request_id
	utils.AssignRequestId(workflow.JobType, workflow)

	// 9. Read rdstore data
	if pipeline.ShouldReadFromRdstore(workflow) {
		// Product url & parent sku checks have to be performed to avoid reading rdstore data
		// for category pages during discovery_crawl requests
		if workflow.DomainInfo.IsProductUrl && workflow.DomainInfo.ParentSku != "" {
			rdstoreData, err := rdutils.FetchParentSKU(url, siteName, parentSku, appC.ConfigData.RestRdstoreUpdate)
			if err != nil {
				utils.FailWorkflow(task, pipeline, workflow, "RDSTORE_READ_FAIL", err.Error(), appC)
				return workflow
			}
			if workflow.JobType == "recrawl" && !rdutils.CheckIfParentSKUFound(rdstoreData) {
				utils.FailWorkflow(task, pipeline, workflow, "RDSTORE_DATA_MISSING_EARLY", fmt.Sprintf("%v", rdstoreData), appC)
				return workflow
			}
			workflow.RdstoreData = rdstoreData
		} else {
			log.Printf("PIPELINE_SKIPRDSTOREREAD: (%s) IsProductUrl: %v, ParentSku: %v\n", url, workflow.DomainInfo.IsProductUrl, workflow.DomainInfo.ParentSku)
		}
	}

	duration := utils.ComputeDuration(start)
	workflow.ProductMetrics.DomainInfo = duration
	log.Printf("PIPELINE_DOMAININFO_TIME: (%s, %s) Retrieving domain info (incl rdstore lookup) took %.2f secs\n", siteName, url, duration)

	// 9. MERGE LOGIC INVOKED HERE

	// 9.1 Create the merge object
	mergeObj := merge.Merge{
		MergeMode:       workflow.JobParams.MergeMode,
		DataSources:     workflow.JobParams.DataSources,
		MergePreference: workflow.JobParams.MergePreference,
	}

	// 9.2 Extract data from multiple sources and merge
	// All supervised, unsupervised & other requests are made here
	code, err = mergeObj.Merge(workflow, pipeline, appC)
	if err != nil {
		utils.FailWorkflow(task, pipeline, workflow, code, err.Error(), appC)
		return workflow
	}

	// 10. Print crawl summary
	workflow.Status = 1
	utils.PrintCrawlSummary(url, workflow)

	// 11. Data translation logic
	if data.ShouldTranslateForJob(workflow, workflow.JobType) {
		data.ApplyTranslation(workflow, appC)
		// } else {
		// 	log.Printf("PIPELINE_TRANSLATE: No Translation params found for %s, %s, %s, Skipping translation.\n", siteName, parentSku, jobType)
	}

	// 14. Execute post crawl ops for different job types
	code, err = pipeline.PostCrawlOps(task, workflow, appC)
	workflow.PostCrawlOpsCalled = true
	if err != nil {
		utils.FailWorkflow(task, pipeline, workflow, code, err.Error(), appC)
		return workflow
	}
	return workflow
}

// CrawlURL - Performs crawl for a given workflow object
// NOTE: Only being maintained for backward compatibility
// REST endpoint `crawl/url` which makes use of this function
// is not actively being used in any of production systems
// All Realtime, Webhooks, Console & Hscodes services are using `crawl/url/simple` instead
func CrawlURL(workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (code string, err error) {
	url := workflow.URL

	mergeObj := merge.Merge{
		DataSources: workflow.JobParams.DataSources,
		MergeMode:   workflow.JobParams.MergeMode,
	}

	code, err = mergeObj.Merge(workflow, pipeline, appC)
	if err != nil {
		utils.FailWorkflow(url, pipeline, workflow, code, err.Error(), appC)
		return code, err
	}

	// Update workflow status to success
	workflow.Status = 1
	utils.PrintCrawlSummary(url, workflow)
	return code, nil
}

// handleMissingKeysFromInput - Updates workflow object with default values
// if any of the important keys are missing
func handleMissingKeysFromInput(workflow *types.CrawlWorkflow, appC *types.Config) {
	// If merge mode has not been sent, use CASCADE as the default mode
	if workflow.JobParams.MergeMode == "" {
		workflow.JobParams.MergeMode = "CASCADE"
	}
	// If data_sources are sent use given data_source
	if len(workflow.JobParams.DataSources) > 0 {
		return
	}
	// set default data_source
	workflow.JobParams.DataSources = []string{"WRAPPER"}
	// check redis map for data_source for given site and given job_type
	domainName := workflow.DomainInfo.DomainName
	// redis key for domain-source map
	redisKey := getDataSourceMapRedisKey(workflow.JobInput.JobDetails.JobType)
	if redisKey == "" {
		return
	}
	dataSources := getDomainDataSource(appC.RedisRdstore, redisKey, domainName)
	if len(dataSources) > 0 {
		workflow.JobParams.DataSources = dataSources
		return
	}
	dataSources = getDomainDataSource(appC.RedisRdstore, redisKey, "default")
	if len(dataSources) > 0 {
		workflow.JobParams.DataSources = dataSources
		return
	}
}

// get redis key for data_source map wrt job type
func getDataSourceMapRedisKey(jobType string) string {
	if strings.Contains(jobType, "webhooks") {
		return "webhooks_domain_source_map"
	} else if jobType == "realtimeapi" {
		return "realtime_domain_source_map"
	}
	return ""
}

// get data source for domain from redis
func getDomainDataSource(rdStore *redis.Pool, redisKey string, domain string) []string {
	var ds []string
	dsRaw, _ := rh.HGet(rdStore, redisKey, domain)
	if dsRaw != "" {
		_ = json.Unmarshal([]byte(dsRaw), &ds)
	}
	return ds
}

// ValidateDomainInfoForSupervised will verify if domainInfo is valid (exclusive for supervised data source)
// If WRAPPER is the only data source (supervised extraction)
// Sitedetails should be present && Site status should be matching whatever pipeline specifies
func ValidateDomainInfoForSupervised(workflow *types.CrawlWorkflow, allowedSiteStatus string) (string, error) {
	extractionMode := utils.GetExtractionMode(workflow.JobParams.DataSources)
	if extractionMode == "WRAPPER" {
		di := workflow.DomainInfo
		site := workflow.DomainInfo.DomainName

		if di == nil || di.Sitedetail == nil {
			return "NO_SITEDETAIL", fmt.Errorf("no sitedetail found for %s", site)
		}
		siteStatus := di.SiteStatus
		allowedSiteStatusRegex, _ := regexp.Compile(allowedSiteStatus)
		if !allowedSiteStatusRegex.MatchString(di.SiteStatus) {
			return "SITE_STATUS_CHECK_FAILED", fmt.Errorf("could not process recrawl req for %s in %s state", site, siteStatus)
		}
	}
	return "", nil
}

// Default validate web response
func DefaultValidateWebResponse(workflow *types.CrawlWorkflow) (canExtract bool, code string, err error) {
	url := workflow.URL
	status := workflow.WebResponse.Status
	jobParams := workflow.JobParams

	switch true {
	// Handle http_200s
	case html.IsSuccess(status):
		canExtract = true
	// Handle http_500s
	case html.IsTempError(status):
		log.Printf("CRAWL_FAILURE: (%s) HTTPStatus %d, Not extracting content\n", url, status)
		code = "HTTP_500_ERROR"
		err = fmt.Errorf("failed to crawl %s: %d", url, status)
	// Handle http_404s
	case html.IsPermError(status):
		if jobParams.ExtractData == 1 {
			canExtract = true
		} else {
			log.Printf("CRAWL_DISC: (%s) HTTPStatus %d, Not extracting content\n", url, status)
		}
	}

	return canExtract, code, err
}

func DefaultPrepareRequestConfig(workflow *types.CrawlWorkflow) (reqConfig types.RequestConfig, code string, err error) {
	url := workflow.URL
	jobType := jobutils.GetJobType(workflow.JobInput)
	siteName := workflow.DomainInfo.DomainName

	reqConfig = utils.ConstructRequestConfig(url, siteName, false, workflow)
	wrapperBrowser := workflow.DomainInfo.Wrapper.Setup.Browser
	cacheId, err := utils.ConstructCacheId(url, siteName, jobType, workflow, wrapperBrowser)
	if err != nil {
		return reqConfig, "CONTENT_ID_GENERATION_FAILED", err
	}

	// Check if any cache folder has been sent in job_params
	cacheFolder := "ce"
	if workflow.JobParams.CacheFolder != "" {
		cacheFolder = workflow.JobParams.CacheFolder
	}

	domainKey := strings.Replace(siteName, ".", "_", -1)
	cacheKey := fmt.Sprintf("%s/%s/%s/%s", cacheFolder, jobType, domainKey, cacheId)
	reqConfig.CacheKey = cacheKey
	reqConfig.CacheFolder = cacheFolder

	if workflow.JobParams.ExtractData == 1 {
		reqConfig.CacheEvent = "on_success_or_perm_error"
	}

	return reqConfig, "", nil
}
