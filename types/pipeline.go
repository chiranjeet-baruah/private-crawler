package types

type (
	// Crawl pipeline interface to be passed to pipeline executor
	Pipeline interface {
		// Post crawl operations for each job type
		PreCrawlOps(task string, workflow *CrawlWorkflow, appC *Config) (url string, code string, err error)

		// should read from rdstore: whether to read from rdstore (job_type dependent)
		ShouldReadFromRdstore(workflow *CrawlWorkflow) bool

		// Time (in seconds) before it gets expired from cache
		GetCacheExpiryTime() int32

		// check site status and sitedetail
		// return an error in case of validation failure and nil otherwise. nil implies that the
		// workflow can continue
		ValidateDomainInfo(workflow *CrawlWorkflow) (code string, err error)

		// check web response (recrawl can add the REDIRECT_SKU_ERROR check here)
		// return an error in case of validation failure and nil otherwise. nil implies that the
		// workflow can continue
		ValidateWebResponse(workflow *CrawlWorkflow) (canExtract bool, code string, err error)

		// check extraction response (recrawl can add the EXTRACTION_FAILED_NOPRODS check here)
		// return an error in case of validation failure and nil otherwise. nil implies that the
		// workflow can continue
		ValidateExtractionResponse(workflow *CrawlWorkflow) (code string, err error)

		// Cache related functions
		PrepareRequestConfig(workflow *CrawlWorkflow) (reqConfig RequestConfig, code string, err error)
		ShouldReadFromCache(workflow *CrawlWorkflow) bool

		// transform error (realtime can use this to rewrite the code and message of internal
		// errors)
		TransformError(string, error) (string, error)

		// Should call post crawl ops on failure as well
		ShouldCallPostCrawlOpsOnFailure(workflow *CrawlWorkflow) bool

		// Post crawl operations for each job type
		PostCrawlOps(task string, workflow *CrawlWorkflow, appC *Config) (code string, err error)
	}
)
