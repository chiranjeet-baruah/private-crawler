package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	htmlutils "github.com/Semantics3/sem3-go-crawl-utils/html"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
)

// VisitPage function handles
// 1. Making request to proxycloud
// 2. Handling client retries
func VisitPage(url string, config *types.RequestConfig, jobParams *ctypes.CrawlJobParams, productMetrics *types.ProductMetrics, appC *types.Config) (webResponse types.WebResponse) {

	curAttempt := 1

	// Read max_attempts from job params
	maxAttempts := 3
	if jobParams.MaxAttempts > 0 {
		maxAttempts = jobParams.MaxAttempts
	}

	// Print information of how long service has been waiting on rpc
	start := time.Now()
	ticker := time.NewTicker(time.Millisecond * 10000)
	var tickerDrones []chan bool
	defer (func() {
		ticker.Stop()
		for _, tickerDone := range tickerDrones {
			tickerDone <- true
		}
	})()

	for curAttempt <= maxAttempts {
		status := webResponse.Status
		if status == 0 || (htmlutils.IsTempError(status) && jobParams.DontRetry == 0) {
			logMessage := fmt.Sprintf("WEBCRAWL_START: Url: %s, IsAjax: %t, Status: %d, CurrentAttempt: %d", url, config.IsAjax, status, curAttempt)

			// Set new start time for requests in subsequent attempts
			if htmlutils.IsTempError(status) {
				start = time.Now()
			}

			utils.PrintResponseDetails(status, logMessage)
			tickerDrone := make(chan bool)
			tickerDrones = append(tickerDrones, tickerDrone)
			go func(u string, tickerDone chan bool) {
				for {
					select {
					case <-tickerDone:
						return
					case <-ticker.C:
						log.Printf("WEBCRAWL_WAIT: Url: %s, IsAjax: %t, ProxyCloudWaitingTime: %f", u, config.IsAjax, utils.ComputeDuration(start))
					}
				}
			}(url, tickerDrone)

			// Request page based on method
			if config.Method != "" && config.Method == "POST" {
				webResponse = PostRequest(url, config, jobParams, appC)
			} else {
				webResponse = GetRequest(url, productMetrics.Site, productMetrics.JobType, config, jobParams, appC)
			}
			webResponse.Attempts = (curAttempt - 1)

			// Make sure ajax calls dont run into race conditions while updating the value
			var m sync.Mutex
			m.Lock()
			if htmlutils.IsTempError(webResponse.Status) {
				utils.CollectProductMetrics("latency", webResponse.TimeTaken, productMetrics)
			}
			if curAttempt == 1 {
				utils.CollectProductMetrics("url_count", 1, productMetrics)
			} else {
				utils.CollectProductMetrics("retry_count", 1, productMetrics)
			}
			m.Unlock()
		}
		if !htmlutils.IsTempError(webResponse.Status) {
			break
		}
		curAttempt++
	}

	return
}

// GetScreenshot makes a simple request to proxycloud for screenshot
func GetScreenshot(appC *types.Config, request types.ScreenshotRequest) (err error) {
	log.Printf("PCREQUEST_START: (%s, %s) Request policy %s", request.URL, request.Domain, request.RequestPolicy)
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}

	router := fmt.Sprintf("http://%s/crawl/url", appC.ConfigData.ProxyRouter)
	req, err := http.NewRequest("POST", router, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	timeout := request.Timeout + 5
	client := &http.Client{
		Timeout: time.Second * time.Duration(timeout),
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("BODY_PARSE_ERR: %v", err)
	}
	var res ctypes.WebResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		return fmt.Errorf("JSON_UNMARSHAL_ERR: %v", err)
	}
	if res.Error != "" || !res.Success {
		return fmt.Errorf("RESPONSE_ERR: code: %d, message: %s", res.Status, res.Message)
	}
	return nil
}
