package utils

import (
	"strconv"

	"github.com/Semantics3/go-crawler/types"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
)

func UpdateCrawlMetrics(site string, config *types.RequestConfig, webResponse *types.WebResponse, jobParams *ctypes.CrawlJobParams, appC *types.Config) {

	var crawlMetrics types.CrawlMetrics

	// Collect input
	crawlMetrics.JobType = config.JobType
	crawlMetrics.Site = config.DomainInfo.DomainName

	// DomainName is an empty string for secondary web requests
	// Hence set manually
	if config.IsAjax == true {
		crawlMetrics.Site = site
	}

	// Identify the customer
	if jobParams.Customer != "" {
		crawlMetrics.Customer = jobParams.Customer
	} else {
		crawlMetrics.Customer = "sem3"
	}

	// Identify the recrawl frequency
	if jobParams.RecrawlFrequency != "" {
		crawlMetrics.RecrawlFrequency = jobParams.RecrawlFrequency
	} else {
		crawlMetrics.RecrawlFrequency = "non_recrawl"
	}

	// Identify request pools
	crawlMetrics.NodePool = webResponse.Headers.XNodePool
	if webResponse.Headers.XRenderPool != "" {
		crawlMetrics.RenderPool = webResponse.Headers.XRenderPool
	} else {
		crawlMetrics.RenderPool = "no_render"
	}

	crawlMetrics.ContentLength = webResponse.ResponseSize
	crawlMetrics.Status = strconv.Itoa(webResponse.Status)
	crawlMetrics.IsAjax = strconv.FormatBool(config.IsAjax)
	crawlMetrics.Latency = webResponse.TimeTaken
	crawlMetrics.Value = 1

	// Send the crawl metrics object to stats manager
	appC.StatsManager.CrawlMetricsChannel <- crawlMetrics
}

func UpdateProductMetrics(workflow *types.CrawlWorkflow, appC *types.Config) {

	// Set default customer
	if workflow.ProductMetrics.Customer == "" {
		workflow.ProductMetrics.Customer = "sem3"
	}

	// Set default frequency
	if workflow.ProductMetrics.RecrawlFrequency == "" {
		workflow.ProductMetrics.RecrawlFrequency = "non_recrawl"
	}

	if workflow.ProductMetrics.Extraction > 0 {
		workflow.ProductMetrics.Success = "true"
	} else {
		workflow.ProductMetrics.Success = "false"
	}

	workflow.ProductMetrics.Value = 1

	// Send the crawl metrics object to stats manager
	PrettyJSON("PRODUCT_METRICS", workflow.ProductMetrics, true)
	appC.StatsManager.ProductMetricsChannel <- workflow.ProductMetrics
}

func CollectProductMetrics(key string, value interface{}, productMetrics *types.ProductMetrics) {
	switch key {
	case "total":
		productMetrics.Total = value.(float64)
	case "extraction":
		productMetrics.Extraction += value.(float64)
	case "latency":
		productMetrics.Latency += value.(float64)
	case "url_count":
		productMetrics.UrlCount += value.(int)
	case "retry_count":
		productMetrics.RetryCount += value.(int)
	case "error_code":
		productMetrics.ErrorCode = value.(string)
	}
}

// CollectExtractionMetrics will parse extraction_metrics from CE response for every iteration
// It then aggragated to extraction metrics at workflow level
// Eg: If we've 3 iterations (back n forth communications between crawler and CE) for a single task (parent url)
// For every iteration CE will send fresh extraction metrics in it's response
// It's crawler's responsibility to add them up to metrics collected at previous iteration
// So the total extraction time will be iteration 1 + iteration 2 + iteration 3
func CollectExtractionMetrics(aggregatedExtractionMetrics *types.ExtractionMetrics, em *types.ExtractionMetrics) {
	aggregatedExtractionMetrics.S3 += em.S3
	aggregatedExtractionMetrics.Products += em.Products
	aggregatedExtractionMetrics.Links += em.Links
	aggregatedExtractionMetrics.Preprocess += em.Preprocess
	aggregatedExtractionMetrics.UrlCount += em.UrlCount
	aggregatedExtractionMetrics.Iterations += 1

	// Compute total
	aggregatedExtractionMetrics.Total = aggregatedExtractionMetrics.S3 + aggregatedExtractionMetrics.Products + aggregatedExtractionMetrics.Links + aggregatedExtractionMetrics.Preprocess
}

// UpdateExtractionMetrics will send extraction metrics for a single task (parent url) to stats aggregator
func UpdateExtractionMetrics(workflow *types.CrawlWorkflow, appC *types.Config) {

	// Assign a customer
	if workflow.ExtractionMetrics.Customer == "" {
		if workflow.JobParams.Customer != "" {
			workflow.ExtractionMetrics.Customer = workflow.JobParams.Customer
		} else {
			workflow.ExtractionMetrics.Customer = "sem3"
		}
	}

	// Assign site name
	if workflow.ExtractionMetrics.Site == "" {
		workflow.ExtractionMetrics.Site = workflow.DomainInfo.DomainName
	}

	// Assign job type
	if workflow.ExtractionMetrics.JobType == "" {
		workflow.ExtractionMetrics.JobType = workflow.ProductMetrics.JobType
		// workflow.ExtractionMetrics.JobType = jobutils.GetJobType(workflow.JobInput)
	}

	workflow.ExtractionMetrics.Value = 1

	// Send the extraction metrics object to stats manager
	appC.StatsManager.ExtractionMetricsChannel <- workflow.ExtractionMetrics
}

// Usage: UpdateJobServerBatchStats(25, workflow, 23.00, appC)
func UpdateJobServerBatchStats(batchSize int, workflow *types.CrawlWorkflow, duration float64, appC *types.Config) {
	appC.StatsManager.JobServerBatchStatsChannel <- types.JobServerBatchStats{
		Site:             workflow.ProductMetrics.Site,
		Customer:         workflow.ProductMetrics.Customer,
		JobType:          workflow.ProductMetrics.JobType,
		RecrawlFrequency: workflow.ProductMetrics.RecrawlFrequency,
		BatchSize:        batchSize,
		Duration:         duration,
		Value:            1,
	}
}
