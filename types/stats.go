package types

import "time"

type (
	StatsManager struct {
		BatchCrawlMetrics          BatchCrawlMetrics        `json:"batch_crawl_metrics"`
		BatchProductMetrics        BatchProductMetrics      `json:"batch_product_metrics"`
		BatchExtractionMetrics     BatchExtractionMetrics   `json:"batch_extraction_metrics"`
		Init                       bool                     `json:"init"`
		LastFlushTime              time.Time                `json:"last_flush_time"`
		CrawlMetricsChannel        chan CrawlMetrics        `json:"crawl_metrics_channel"`
		ProductMetricsChannel      chan ProductMetrics      `json:"product_metrics_channel"`
		ExtractionMetricsChannel   chan ExtractionMetrics   `json:"extraction_metrics_channel"`
		JobServerBatchStatsChannel chan JobServerBatchStats `json:"job_server_batch_stats_channel"`
	}

	CrawlMetrics struct {
		Site string `json:"site"`
		// Default to sem3
		Customer string `json:"customer"`
		JobType  string `json:"job_type"`
		// Defaults to no_recrawl
		RecrawlFrequency string `json:"recrawl_frequency"`
		// Proxy pool that was used for the request
		NodePool string `json:"node_pool"`
		// Render pool that was used for the request
		RenderPool string `json:"render_pool"`
		// Web response status code as a string
		Status string `json:"status"`
		// True indicates a secondary web request
		IsAjax string `json:"is_ajax"`
		// Length of the web response content
		ContentLength int     `json:"content_length"`
		Latency       float64 `json:"latency"`
		// Always 1
		Value int `json:"value"`
	}

	BatchCrawlMetrics map[string]map[string]map[string]map[string]map[string]map[string]map[string]map[string]map[string]interface{}

	ProductMetrics struct {
		Site string `json:"site"`
		// Defaults to sem3
		Customer string `json:"customer"`
		JobType  string `json:"job_type"`
		// Defaults to non_recrawl
		RecrawlFrequency string `json:"recrawl_frequency"`
		// Extraction time > 0.0 => Product Success
		Success string `json:"success"`
		// Total time taken by the Content Extraction service to handle content extraction for this product
		// This includes primary and secondary requests
		Extraction float64 `json:"extraction"`
		// Total = (Latency + DomainInfo + Extraction)
		Total float64 `json:"total"`
		// Crawl latency (Sum of latencies for crawling primary and secondary web requests including retries)
		Latency float64 `json:"latency"`
		// Number of URLs crawled for this product - this includes primary and secondary web requests
		UrlCount int `json:"url_count"`
		// Sum of all crawl retries performed including the primary and secondary web requests (worst-cast: 3 * UrlCount)
		RetryCount int `json:"retry_count"`
		// Time taken for fetching domain information (Redis Wrapper/Mongo Wrapper/RD Store)
		DomainInfo float64 `json:"domain_info"`
		// Error Code during failures
		ErrorCode string `json:"error_code"`
		// Always 1
		Value int `json:"value"`
	}

	BatchProductMetrics map[string]map[string]map[string]map[string]map[string]map[string]interface{}

	ExtractionMetrics struct {
		Site string `json:"site"`
		// Defaults to sem3
		Customer string `json:"customer"`
		JobType  string `json:"job_type"`
		// Time taken for s3 latencies (for both ajax and primary)
		S3 float64 `json:"s3"`
		// Time taken for pure products extraction (ajax latencies are deducted and added to s3)
		Products float64 `json:"products"`
		// Time taken for extracting links
		Links float64 `json:"links"`
		// Time taken for executing preprocess script
		// Ajax latencies during execuction are deducted here and added to s3
		Preprocess float64 `json:"preprocess"`
		// Total (s3+ products + links + preprocess)
		Total float64 `json:"total"`
		// Total # of urls (primary + ajax)
		UrlCount int `json:"url_count"`
		// # of iterations made back&forth between crawler and ce
		Iterations int `json:"iterations"`
		// Always 1
		Value int `json:"value"`
	}

	BatchExtractionMetrics map[string]map[string]map[string]map[string]interface{}

	JobServerBatchStats struct {
		Site             string  `json:"site"`
		Customer         string  `json:"customer"`
		JobType          string  `json:"job_type"`
		RecrawlFrequency string  `json:"recrawl_frequency"`
		Duration         float64 `json:"duration"`
		BatchSize        int     `json:"batch_size"`
		Value            int     `json:"value"`
	}
)
