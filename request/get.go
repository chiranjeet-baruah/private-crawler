package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	htmlutils "github.com/Semantics3/sem3-go-crawl-utils/html"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

type pRequest ctypes.WebRequest
type pResponse ctypes.WebResponse

var crawleraRequestPolicy *regexp.Regexp
var renderingEngineRegex *regexp.Regexp

// Handles the following things
// 1. Constructing the request payload
// 2. Downloading web page by requesting proxycloud
// 3. Copying the proxycloud response to crawl workflow response
// 4. Updating crawl metrics
func GetRequest(url, site, jobType string, config *types.RequestConfig, jobParams *ctypes.CrawlJobParams, appC *types.Config) (webResponse types.WebResponse) {

	var request pRequest
	var response pResponse

	request.Headers = make(map[string]string)

	start := time.Now()

	// 1. Construct request payload
	request.constructPayload(url, jobType, config, jobParams, appC)

	// 2. Make request to proxycloud
	request.fetchPage(&response, appC)

	// 3. Copy response
	response.CopyResponse(url, &webResponse, utils.ComputeDuration(start), config)

	// 4. Update crawl metrics to influx
	utils.UpdateCrawlMetrics(site, config, &webResponse, jobParams, appC)

	// NOTE: Debug information needed for rendering through crawlera
	if webResponse.Headers.XNodePool == "crawlera_exclusive" && webResponse.Headers.XRenderPool != "" {
		msg1 := fmt.Sprintf("REQUEST_DETAILS: URL: %s, REQUEST: ", url)
		utils.PrettyJSON(msg1, request, true)
		msg2 := fmt.Sprintf("REQUEST_DETAILS: URL: %s, JOB_PARAMS: ", url)
		utils.PrettyJSON(msg2, jobParams, true)
	}

	return
}

// Collects all the request configs from job params and wrapper browser
// Constructs proxycloud request payload
func (request *pRequest) constructPayload(url, jobType string, config *types.RequestConfig, jobParams *ctypes.CrawlJobParams, appC *types.Config) {
	// Retrieve site name
	var site string
	if site = config.DomainInfo.DomainName; site == "" {
		var err error
		domain, err := utils.GetDomainName(url, appC.ConfigData.WrapperServiceURI)
		if err != nil {
			log.Fatalln(err)
		}

		if domain != "" {
			site = domain
		}
	}

	wrapperBrowser := config.DomainInfo.Wrapper.Setup.Browser

	request.URL = strings.TrimSuffix(url, "\n")
	request.Domain = site
	request.IsAjax = config.IsAjax
	if jobParams.PriorityRequest == 1 {
		request.Priority = true
	}
	request.Tag = config.JobType
	request.Crumb = config.Crumb
	request.Headers = utils.GetRequestHeaders(config, wrapperBrowser)
	request.PageTransforms = utils.GetPageTransforms(&config.DomainInfo.Wrapper)
	request.Pools = utils.GetProxyPools(jobParams, wrapperBrowser)
	request.Sleep = utils.GetSleepTime(site, jobParams, wrapperBrowser)
	request.Timeout = utils.GetRequestTimeout(jobParams, wrapperBrowser)
	request.RequestPolicy = utils.GetRequestPolicy(jobParams, wrapperBrowser)
	request.Cookie = utils.GetCookies(request.RequestPolicy, config, jobParams, wrapperBrowser)

	// Handle cases where render param has to be removed for crawlera requests
	pools := request.Pools
	if len(pools) > 0 {
		poolName := "crawlera_exclusive"
		if cutils.StringInSlice(poolName, pools) {
			rand.Seed(time.Now().UnixNano())
			randVal := rand.Intn(100)
			updatedPools := []string{}
			if randVal < 50 {
				for _, p := range pools {
					if p != poolName {
						updatedPools = append(updatedPools, p)
					}
				}
			} else {
				// Remove render component
				if crawleraRequestPolicy == nil {
					crawleraRequestPolicy = regexp.MustCompile(`(?i)render:\d+;?`)
				}
				request.RequestPolicy = crawleraRequestPolicy.ReplaceAllString(request.RequestPolicy, "")

				// Remove rendering engine
				if renderingEngineRegex == nil {
					renderingEngineRegex = regexp.MustCompile(`(?i)rendering_engine:\w+;?`)
				}
				request.RequestPolicy = renderingEngineRegex.ReplaceAllString(request.RequestPolicy, "")
				updatedPools = append(updatedPools, poolName)
			}
			request.Pools = updatedPools
			log.Printf("REQUEST: Random value: %d, Request policy: %s, Pools: %v\n", randVal, request.RequestPolicy, request.Pools)
		}
	}

	// Add cache config to request policy
	if config.CacheKey != "" {
		request.RequestPolicy = fmt.Sprintf("%scache_key:%s;cache_expiry:%d;", request.RequestPolicy, config.CacheKey, config.CacheExpiry)
	}
	if config.CacheEvent != "" {
		request.RequestPolicy = fmt.Sprintf("%scache_event:%s;", request.RequestPolicy, config.CacheEvent)
	}

	// Check if request requires a screenshot
	// DO NOT blanket block ajax calls as pages like /gp/offer-listing/ will be lost
	// Usually CE workflow will construct a new request_policy without render in it
	requestPolicy, render := strings.ToUpper(request.RequestPolicy), strings.ToUpper("render:1")
	if jobParams.Screenshot == 1 && strings.Contains(requestPolicy, render) {
		isProductUrl := config.DomainInfo.IsProductUrl
		request.RequestPolicy = utils.GetScreenshotPath(site, url, isProductUrl, request.RequestPolicy, config, jobParams)
	}

	// utils.PrettyJSON("REQUEST_1202: PAYLOAD: ", request, false)
}

// Handle proxycloud (http) request/response
func (request *pRequest) fetchPage(response *pResponse, appC *types.Config) {
	log.Printf("PCREQUEST_START: (%s, %s) Request policy %s", request.URL, request.Domain, request.RequestPolicy)
	payload, err := json.Marshal(request)
	if err != nil {
		log.Fatalln(err)
	}

	router := fmt.Sprintf("http://%s/crawl/url", appC.ConfigData.ProxyRouter)
	req, err := http.NewRequest("POST", router, bytes.NewBuffer(payload))
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Set("Content-Type", "application/json")

	timeout := request.Timeout + 5
	client := &http.Client{
		Timeout: time.Second * time.Duration(timeout),
	}

	resp, err := client.Do(req)
	if err != nil {
		response.handleError(request.URL, err.Error())
		return
	}

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(response)
	if err != nil || response.Error != "" {
		var message string
		if message = response.Error; message == "" {
			message = err.Error()
		}
		response.handleError(request.URL, message)
		return
	}

	// Parse response headers
	response.getHeaders(resp)

	// Get request url
	var url string
	if url = response.RedirectURL; url == "" {
		url = request.URL
	}
}

// Handles errors like
// 1. Malformed proxycloud request payload
// 2. Proxycloud request timeouts
func (response *pResponse) handleError(url string, message string) {
	statusCode := http.StatusInternalServerError

	badrequest := regexp.MustCompile(`invalid json request`)

	if badrequest.MatchString(message) {
		statusCode = http.StatusBadRequest
		message = fmt.Sprintf("Bad Request: %s", message)
	} else if htmlutils.IsRequestTimeout(message) {
		statusCode = http.StatusRequestTimeout
		message = fmt.Sprintf("Request timed out for %s", url)
	}

	logMessage := fmt.Sprintf("PCREQUEST_FAILED: (%s) failed: %s", url, message)
	utils.PrintResponseDetails(500, logMessage)

	response.StatusCode = statusCode
	response.Error = message
}

// Parse headers from proxycloud response
func (response *pResponse) getHeaders(resp *http.Response) {

	var headers ctypes.ResponseHeaders

	// Get proxy node information
	for name, value := range resp.Header {
		switch name {
		case "X-Node-Ip":
			headers.XNodeIp = value[0]
		case "X-Node-Pool":
			headers.XNodePool = value[0]
		case "X-Render-Ip":
			headers.XRenderIp = value[0]
		case "X-Render-Pool":
			headers.XRenderPool = value[0]
		}
	}
	headers.ContentLength = len(response.Content)
	response.Headers = headers
}

// Copy data from proxycloud response to crawl workflow response
func (response *pResponse) CopyResponse(url string, webResponse *types.WebResponse, duration float64, config *types.RequestConfig) {
	webResponse.Status = response.StatusCode
	webResponse.Success = response.Success
	webResponse.TimeTaken = response.TimeTaken
	webResponse.Headers = response.Headers
	webResponse.Content = response.Content

	cLength := 0
	var err error
	if response.Content != "" {
		// Content will be written to S3 and response size is sent through content
		// Parse the size from string and set empty content
		cLength, err = utils.GetContentLength(response.Content)
		if err != nil {
			log.Fatalln(err)
		}
	}
	webResponse.ResponseSize = cLength

	if webResponse.URL = response.URL; webResponse.URL == "" {
		webResponse.URL = url
	}

	if webResponse.Redirect = response.RedirectURL; webResponse.Redirect == "" {
		webResponse.Redirect = url
	}

	// Always take duration as latency for the request
	// This also handles request timeout cases
	webResponse.TimeTaken = duration
	if response.TimeTaken == 0 {
		response.TimeTaken = duration
	}

	// Copy cookie from the response
	if response.Cookie != "" {
		webResponse.Cookie = response.Cookie
	}

	// Copy screenshot path from the request configuration
	if config.ScreenshotPath != "" {
		webResponse.ScreenshotPath = []string{config.ScreenshotPath}
	}

	logMessage := fmt.Sprintf("PCREQUEST_DONE: URL: %s, Status: %d, Latency: %f, Content Size: %d, Round-Trip: %.2f", webResponse.URL, webResponse.Status, response.TimeTaken, cLength, duration)
	utils.PrintResponseDetails(response.StatusCode, logMessage)
	if htmlutils.IsTempError(webResponse.Status) {
		utils.PrettyJSON("REQUEST_HEADERS", webResponse.Headers, true)
	}
}
