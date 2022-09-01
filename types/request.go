package types

import (
	"net/http"

	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
)

type (
	CacheKeyConfig struct {
		Url           string            `json:"url,omitempty"`
		Domain        string            `json:"domain,omitempty"`
		RequestPolicy string            `json:"request_policy,omitempty"`
		Cookie        string            `json:"cookie,omitempty"`
		RequestId     string            `json:"request_id,omitempty"`
		Headers       map[string]string `json:"headers,omitempty"`
	}

	RequestConfig struct {
		DomainInfo     *ctypes.DomainInfo `json:"domainInfo"`
		ProductMetrics ProductMetrics     `json:"product_metrics"`
		JobType        string             `json:"job_type"`
		Crumb          string             `json:"crumb"`
		ParentUrl      string             `json:"parent_url,omitempty"`
		Cookie         string             `json:"cookie,omitempty"`
		CacheKey       string             `json:"cache_key,omitempty"`
		CacheFolder    string             `json:"cache_folder,omitempty"`
		// one of `on_success` OR `on_success_or_temp_error`
		CacheEvent     string `json:"cache_event,omitempty"`
		CacheExpiry    int32  `json:"cache_expiry,omitempty"`
		IsAjax         bool   `json:"is_ajax"`
		IsRetry        bool   `json:"is_retry"`
		ScreenshotPath string `json:"screenshot_path"`

		// Post request specific
		Method  string            `json:"method,omitempty"`
		Headers map[string]string `json:"headers,omitempty"`
		Body    string            `json:"payload,omitempty"`
		Timeout int               `json:"timeout,omitempty"`
	}

	// WebResponse represents the response from the executing the WebRequest
	WebResponse struct {
		URL            string                 `json:"url"`
		Redirect       string                 `json:"redirect"`
		Message        string                 `json:"message,omitempty"`
		Content        string                 `json:"content"`
		Time           int64                  `json:"time"`
		Error          string                 `json:"error,omitempty"`
		Cookie         string                 `json:"cookie,omitempty"`
		FromCache      bool                   `json:"from_cache"`
		Success        bool                   `json:"success"`
		ResponseSize   int                    `json:"response_size"`
		Status         int                    `json:"status"`
		Attempts       int                    `json:"attempts"`
		TimeTaken      float64                `json:"timeTaken"`
		ScreenshotPath []string               `json:"screenshot_path"`
		Headers        ctypes.ResponseHeaders `json:"response_headers"`
	}

	AjaxURL struct {
		URL           string                 `json:"url"`
		AjaxJobParams *ctypes.CrawlJobParams `json:"ajax_job_params,omitempty"`
		CacheKey      string                 `json:"cache_key,omitempty"`

		// Post request specific
		Method  string            `json:"method,omitempty"`
		Headers map[string]string `json:"headers,omitempty"`
		Body    string            `json:"body,omitempty"`
		Cookie  string            `json:"cookie,omitempty"`
		Timeout int               `json:"timeout,omitempty"`
	}

	PostRequestResponse struct {
		URL          string      `json:"url"`
		Content      string      `json:"content"`
		Error        string      `json:"error,omitempty"`
		Cookie       string      `json:"cookie,omitempty"`
		ResponseSize int         `json:"response_size"`
		Status       int         `json:"status"`
		TimeTaken    float64     `json:"timeTaken"`
		Headers      http.Header `json:"response_headers"`
	}

	ScreenshotRequest struct {
		URL           string   `json:"url"`
		Pools         []string `json:"pools"`
		Domain        string   `json:"domain"`
		RequestPolicy string   `json:"request_policy"`
		Timeout       int      `json:"timeout"`
	}

	ScreenshotResponse struct {
		Status         int    `json:"status"`
		Screenshot     string `json:"screenshot"`
		FailureType    string `json:"failuretype,omitempty"`
		FailureMessage string `json:"failuremessage,omitempty"`
	}
)
