package types

import ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"

type (
	ExtractionResponse struct {
		Status                      int                           `json:"status"`
		Code                        string                        `json:"code"`
		Message                     string                        `json:"message,omitempty"`
		ExtractionTime              float64                       `json:"extraction_time,omitempty"`
		ExtractionEngine            string                        `json:"extraction_engine,omitempty"`
		ExtractionDataSource        string                        `json:"extraction_data_source,omitempty"`
		Products                    []map[string]interface{}      `json:"products"`
		Categories                  []map[string]interface{}      `json:"categories,omitempty"`
		Cart                        []map[string]interface{}      `json:"cart,omitempty"`
		Links                       map[string]ctypes.UrlMetadata `json:"links,omitempty"`
		SpideringLinks              []map[string]interface{}      `json:"spidering_links"`
		ExtractionBreakdown         map[string]interface{}        `json:"extraction_breakdown,omitempty"`
		ExtractionMetrics           map[string]interface{}        `json:"extraction_metrics,omitempty"`
		RawExtractedData            map[string]interface{}        `json:"__raw_extracted_data,omitempty"`
		UnresolvedAjaxURLs          []AjaxURL                     `json:"unresolved_ajax_urls,omitempty"`
		OverridingWebResponseStatus int                           `json:"overriding_webresponse_status"`
		WrapperFilterResults        map[string]bool               `json:"wrapper_filter_results"`
	}

	UnsupervisedResponse struct {
		Time           int64                  `json:"time,omitempty"`
		Status         int                    `json:"status,omitempty"`
		Error          string                 `json:"error,omitempty"`
		URL            string                 `json:"url,omitempty"`
		Message        string                 `json:"message,omitempty"`
		Html           string                 `json:"html,omitempty"`
		Content        string                 `json:"content,omitempty"`
		Image          string                 `json:"image,omitempty"`
		XUceIP         string                 `json:"x-uce-ip,omitempty"`
		ScreenshotPath string                 `json:"screenshot_path,omitempty"`
		Result         map[string]interface{} `json:"result,omitempty"`
	}

	Data struct {
		Products   []map[string]interface{} `json:"products"`
		Categories []map[string]interface{} `json:"categories,omitempty"`
		Cart       []map[string]interface{} `json:"cart,omitempty"`
	}
)
