package types

import (
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
)

type (
	CrawlWorkflow struct {
		URL                 string                   `json:"url"`
		RequestId           string                   `json:"request_id,omitempty"`
		JobType             string                   `json:"job_type,omitempty"`
		JobInput            *ctypes.Batch            `json:"jobInput,omitempty"`
		JobParams           *ctypes.CrawlJobParams   `json:"job_params"`
		DomainInfo          *ctypes.DomainInfo       `json:"domainInfo,omitempty"`
		RdstoreData         *ctypes.RdstoreParentSKU `json:"rdstore_data,omitempty"`
		WebResponse         WebResponse              `json:"webResponse"`
		AjaxFailedStatusMap map[string]int           `json:"ajax_failed_status_map"`
		Data                ExtractionResponse       `json:"data"`
		ProductMetrics      ProductMetrics           `json:"product_metrics"`
		ExtractionMetrics   ExtractionMetrics        `json:"extraction_metrics"`

		// Translate crawl related cache, stats
		IsTranslateCrawl bool                              `json:"is_translate_crawl,omitempty"`
		RawData          map[string]map[string]interface{} `json:"raw_data,omitempty"` //NOTE: Skus DB Record.

		CacheKey             string `json:"cache_key"`
		CacheExpiry          int32  `json:"cache_expiry"`
		UnsupervisedCacheKey string `json:"unsupervised_cache_key"`
		CrawlTime            int64  `json:"crawl_time"`

		Status         int     `json:"status"`
		FailureType    *string `json:"failuretype"`
		FailureMessage *string `json:"failuremessage"`

		// Fields exclusive to "consumer" mode
		QueueName string

		ValidateErrors        *ctypes.ValidateErrs `json:"validation"`
		PostCrawlOpsCalled    bool                 `json:"post_crawl_ops_called"`
		PreCrawlOpsFailed     bool                 `json:"pre_crawl_ops_failed"`
		SendFailureAsFeedback bool                 `json:"send_failure_as_feedback"`
	}

	ForwardedCrawlWorkflow struct {
		DomainInfo  *ctypes.DomainInfo `json:"domainInfo,omitempty"`
		Data        ExtractionResponse `json:"data"`
		WebResponse WebResponse        `json:"webResponse"`
	}

	RdstoreResp struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}

	SpideringOutput struct {
		JobID                string `json:"job_id"`
		Site                 string `json:"site"`
		CreatedAt            int64  `json:"created_at"`
		ParentLink           string `json:"parent_link"`
		TotalLinks           int    `json:"total_links"`
		CategoryLinks        int    `json:"category_links"`
		SitemapLinks         int    `json:"sitemap_links"`
		ProductLinks         int    `json:"product_links"`
		ProductLinksFiltered int    `json:"product_links_filtered"`
		SkippedLinks         int    `json:"skipped_links"`
	}

	JobServerFeedback struct {
		Metadata ctypes.UrlMetadata `json:"metadata"`
		Priority int                `json:"priority"`
	}
)
