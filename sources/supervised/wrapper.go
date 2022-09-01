package supervised

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Semantics3/go-crawler/request"
	"github.com/Semantics3/go-crawler/sources"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	"github.com/Semantics3/sem3-go-crawl-utils/html"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

func extractDataUsingWrapper(url string, workflow *types.CrawlWorkflow, appC *types.Config) (err error, errCode string) {

	// Perform validation
	if workflow.DomainInfo.WrapperId == "" {
		return errors.New("cannot perform supervised extraction since the wrapper is empty"), "WRAPPER_EMPTY"
	}

	if workflow.JobParams.ExtractIgnoreSite == 0 {

		if workflow.DomainInfo.Sitedetail == nil {
			return errors.New("cannot perform supervised extraction since the sitedetail is empty"), "SITEDETAIL_EMPTY"
		}

		if workflow.DomainInfo.SiteStatus == "" {
			return fmt.Errorf("site status cannot be found for %s", workflow.DomainInfo.DomainName), "SITE_STATUS_CHECK_FAILED_NOT_FOUND"
		}

		if ok, _ := cutils.ApplyRegex("siteStatus", workflow.DomainInfo.SiteStatus, "stoppedSite", `STOP_|DELETED`, ""); ok {
			return fmt.Errorf("site %s is in the %s status", workflow.DomainInfo.DomainName, workflow.DomainInfo.SiteStatus),
				"SITE_STATUS_CHECK_FAILED_DELETED_SITE"
		}

		if ok, _ := cutils.ApplyRegex("siteStatus", workflow.DomainInfo.SiteStatus, "brokenSite", `PAUSE_BROKEN_WRAPPER|PAUSE_MAINTENANCE`, ""); ok {
			return fmt.Errorf("site %s is in the %s status", workflow.DomainInfo.DomainName, workflow.DomainInfo.SiteStatus),
				"SITE_STATUS_CHECK_FAILED_BROKEN_WRAPPER"
		}

	}

	// Collect extraction duration for product metrics
	start := time.Now()
	defer func(begin *time.Time) {
		utils.CollectProductMetrics("extraction", utils.ComputeDuration(*begin), &workflow.ProductMetrics)
	}(&start)

	workflow.Data = types.ExtractionResponse{}
	args := []interface{}{workflow}
	err = sources.MakeRPCRequest(appC.RPCClient, "WRAPPER", url, "extractWithWrapper", args, &workflow.Data)
	if err != nil {
		return err, ""
	}

	if workflow.Data.Status == 0 {
		return errors.New(workflow.Data.Message), ""
	}

	if workflow.Data.OverridingWebResponseStatus > 0 {
		workflow.WebResponse.Status = workflow.Data.OverridingWebResponseStatus
		workflow.WebResponse.Success = html.IsSuccess(workflow.WebResponse.Status)
	}

	if workflow.Data.WrapperFilterResults != nil {
		isProductUrl, ok := workflow.Data.WrapperFilterResults["products"]
		if ok {
			log.Printf("EXTRACTION_RESPONSE_OVERRIDE_PRODUCTURL: (%s) isProductUrl before %v, after %v\n", url, workflow.DomainInfo.IsProductUrl, isProductUrl)
			workflow.DomainInfo.IsProductUrl = isProductUrl
		}
	}

	// Aggregate extraction metrics for every iteration
	em := &types.ExtractionMetrics{}
	extractionMetricsJson, err := json.Marshal(workflow.Data.ExtractionMetrics)
	if err != nil {
		log.Printf("EXTRACTION_METRICS: Marshalling extraction metrics failed with error: %s: %v, %v\n", url, workflow.Data.ExtractionMetrics, err)
	}
	if err := json.Unmarshal(extractionMetricsJson, em); err != nil {
		log.Printf("EXTRACTION_METRICS: Unmarshalling extraction metrics failed with error: %s: %s, %v\n", url, string(extractionMetricsJson), err)
	}

	utils.CollectExtractionMetrics(&workflow.ExtractionMetrics, em)

	return nil, ""
}

func ExtractDataForAjaxRequests(url string, workflow *types.CrawlWorkflow, appC *types.Config) (code string, err error) {
	jobParams := workflow.JobParams
	iteration := 1
	workflow.AjaxFailedStatusMap = make(map[string]int)

	for len(workflow.Data.UnresolvedAjaxURLs) > 0 {
		index := 0
		start := time.Now()
		numAjaxRequests := len(workflow.Data.UnresolvedAjaxURLs)
		ajaxResponseChan := make(chan func() (types.WebResponse, types.AjaxURL), numAjaxRequests)
		log.Printf("CRAWL_AJAX_START: (%s), ITERATION: %d, AJAX_REQUESTS_COUNT: %d\n", url, iteration, numAjaxRequests)

		// Visit all the ajax calls concurrently and pass them through channel
		// 1. To handle race conditions while updating product metrics
		// 2. To make sure all responses are collected before proceeding to extraction
		for index < numAjaxRequests {
			go func(counter int) {
				var requestConfig types.RequestConfig
				var ajaxJobParams = jobParams
				ajaxConfig := workflow.Data.UnresolvedAjaxURLs[counter]
				if ajaxConfig.AjaxJobParams != nil {
					ajaxJobParams = ajaxConfig.AjaxJobParams
				}

				var cookie string
				if ajaxConfig.Cookie != "" {
					cookie = ajaxConfig.Cookie
				} else {
					cookie = workflow.WebResponse.Cookie
				}

				requestConfig = types.RequestConfig{
					JobType: workflow.ProductMetrics.JobType,
					// We cannot use the parent domain info here even though it may seem intuitive to
					// do so because a secondary web request can be made to a different domain (eg. Grainger)
					// If we set the domain to the parent's, all secondary Proxy Cloud requests may fail since
					// the domain would be seen as having changed.
					DomainInfo:     &ctypes.DomainInfo{},
					ProductMetrics: workflow.ProductMetrics,
					IsAjax:         true,
					ParentUrl:      url,
					Cookie:         cookie,
					CacheKey:       ajaxConfig.CacheKey,
					CacheExpiry:    workflow.CacheExpiry,
					Method:         ajaxConfig.Method,
					Body:           ajaxConfig.Body,
					Headers:        ajaxConfig.Headers,
					Timeout:        ajaxConfig.Timeout,
				}

				counter++
				logMessage := fmt.Sprintf("CRAWL_AJAX_URL_START: AJAX_REQUEST_COUNT: (%d/%d), URL: %s, AJAX_URL: %s\n", counter, numAjaxRequests, url, ajaxConfig.URL)
				utils.PrintResponseDetails(0, logMessage)
				webResponse := request.VisitPage(ajaxConfig.URL, &requestConfig, ajaxJobParams, &workflow.ProductMetrics, appC)
				if !html.IsSuccess(webResponse.Status) {
					workflow.AjaxFailedStatusMap[ajaxConfig.CacheKey] = webResponse.Status
				}
				ajaxResponseChan <- (func() (types.WebResponse, types.AjaxURL) {
					return webResponse, ajaxConfig
				})
			}(index)
			index++
		}

		numAjaxResponse := 1
		for numAjaxResponse <= numAjaxRequests {
			// Collect ajax responses
			webResponse, _ := (<-ajaxResponseChan)()
			utils.CollectScreenshots(workflow, webResponse)
			logMessage := fmt.Sprintf("CRAWL_AJAX_URL_END: AJAX_RESPONSE_COUNT: (%d/%d), URL: %s, AJAX_URL: %s\n", numAjaxResponse, numAjaxRequests, url, webResponse.URL)
			utils.PrintResponseDetails(webResponse.Status, logMessage)
			numAjaxResponse++
		}

		// Close the channel to communicate completion to channel's senders/receivers
		close(ajaxResponseChan)

		duration := utils.ComputeDuration(start)
		utils.CollectProductMetrics("latency", duration, &workflow.ProductMetrics)

		// Extract data
		err, errCode := extractDataUsingWrapper(url, workflow, appC)
		if err != nil {
			if errCode != "" {
				errCode = "EXTRACTION_AJAX_" + errCode
			} else {
				errCode = "EXTRACTION_AJAX_FAILED"
			}
			return errCode, err
		}

		iteration++

		// Make sure we don't into indefinete loop
		if iteration >= 20 {
			code = "EXTRACTION_MAX_CYCLES_EXCEEDED"
			err = cutils.PrintErr(code, fmt.Sprintf("Marking %s as failed as we've reached max extraction cycles limit for a single task", workflow.WebResponse.URL), "")
			return code, err
		}
	}
	return code, nil
}

func DefaultValidateExtractionResponse(workflow *types.CrawlWorkflow) (code string, err error) {
	url := workflow.URL
	status := workflow.WebResponse.Status

	// Ajax response can over-write workflow status (in cases like ajax_important)

	switch true {
	// Handle http_500s
	case html.IsTempError(status):
		log.Printf("CRAWL_FAILURE: (%s) HTTPStatus %d, Not extracting content\n", url, status)
		code = "HTTP_500_ERROR"
		err = fmt.Errorf("failed to crawl %s: %d", url, status)
	// Handle http_404s
	case html.IsPermError(status):
		if workflow.JobParams.ExtractData == 0 {
			log.Printf("EXTRACTION_RESPONSE_FAILED: URL: %s, STATUS: %d, ACTION: Marking the products array as empty\n", url, status)
			workflow.Data.Products = make([]map[string]interface{}, 0)
		}
	}

	return
}
