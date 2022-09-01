package utils

import (
	"crypto/md5"
	b64 "encoding/base64"
	"fmt"
	"io"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Semantics3/go-crawler/types"
	htmlutils "github.com/Semantics3/sem3-go-crawl-utils/html"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
	"github.com/fatih/color"
)

// Get request policy
func GetRequestPolicy(jobParams *ctypes.CrawlJobParams, wrapperBrowser ctypes.WrapperBrowser) (requestPolicy string) {

	if jobParams.RequestPolicy != "" {
		requestPolicy = jobParams.RequestPolicy
	} else if wrapperBrowser.RequestPolicy != "" {
		requestPolicy = wrapperBrowser.RequestPolicy
	}

	r, _ := regexp.Compile("^.*;$")
	if requestPolicy != "" && !r.MatchString(requestPolicy) {
		requestPolicy = fmt.Sprintf("%s;", requestPolicy)
	}

	return
}

// screenshots := config.ScreenshotPath
// screenshots = append(screenshots, screenshotPath)
// config.ScreenshotPath = removeDuplicates(screenshots)

func GetScreenshotPath(site string, url string, isProductUrl bool, requestPolicy string, config *types.RequestConfig, jobParams *ctypes.CrawlJobParams) string {
	customer := "default"
	if jobParams.Customer != "" {
		customer = strings.ToLower(jobParams.Customer)
	}
	screenshotPath := constructScreenshotPath(site, url, isProductUrl, requestPolicy, customer)
	config.ScreenshotPath = screenshotPath
	b64EncodedScreenshotPath := b64.StdEncoding.EncodeToString([]byte(screenshotPath))
	if requestPolicy[len(requestPolicy)-1:] == ";" {
		requestPolicy = fmt.Sprintf("%sscreenshot:%s;", requestPolicy, b64EncodedScreenshotPath)
	} else {
		requestPolicy = fmt.Sprintf("%s;screenshot:%s;", requestPolicy, b64EncodedScreenshotPath)
	}
	return requestPolicy
}

// Construct screenshot path for a request
func constructScreenshotPath(site string, url string, isProductUrl bool, requestPolicy string, customer string) (screenshotPath string) {

	timeString := time.Now().Format("01-02-2006")
	entities := strings.Split(timeString, "-")
	month := entities[0]
	day := entities[1]
	year := entities[2]
	dateString := fmt.Sprintf("%s%s/%s/%d", year, month, day, time.Now().Hour())
	var pageType string

	// addToCart, _ := regexp.Compile("add_to_cart")
	// offers, _ := regexp.Compile("/gp/offer-listing/")

	if strings.Contains(requestPolicy, "add_to_cart") {
		pageType = "add_to_cart"
	} else if site == "amazon.com" && strings.Contains(url, "/gp/offer-listing/") {
		pageType = "offers"
	} else if isProductUrl {
		pageType = "product"
	} else {
		pageType = "category"
	}

	// TODO: Catch error
	hashDigest, _ := Md5Hash(url)
	screenshotPath = fmt.Sprintf("s3://sem3-web-prod/screenshots/%s/%s/%s/%s/", customer, dateString, site, pageType)
	screenshotPath = fmt.Sprintf("%s%s", screenshotPath, hashDigest)
	log.Printf("SITE: %s, URL: %s, PRODUCT_URL: %t, SCREENSHOT_PATH: %s\n", site, url, isProductUrl, screenshotPath)

	return screenshotPath
}

// Set cookies
func GetCookies(requestPolicy string, config *types.RequestConfig, jobParams *ctypes.CrawlJobParams, wrapperBrowser ctypes.WrapperBrowser) (cookie string) {

	if config != nil && config.IsAjax == true && config.Cookie != "" {
		// For ajax requests, set cookie received from parent request
		cookie = config.Cookie
	} else if wrapperBrowser.Cookie == "1" && requestPolicy == "" {
		cookie = "SESSION:2;"
	} else if wrapperBrowser.Cookie != "" && wrapperBrowser.Cookie != "1" {
		cookie = wrapperBrowser.Cookie
	}

	return
}

// Get request timeout
func GetRequestTimeout(jobParams *ctypes.CrawlJobParams, wrapperBrowser ctypes.WrapperBrowser) (timeout int) {
	timeout = 60
	if jobParams.Timeout != 0 {
		timeout = jobParams.Timeout
	} else if wrapperBrowser.Timeout != 0 {
		timeout = wrapperBrowser.Timeout
	}
	return
}

// Get sleep time for the request
func GetSleepTime(site string, jobParams *ctypes.CrawlJobParams, wrapperBrowser ctypes.WrapperBrowser) (sleep int) {

	sleep = 1
	if wrapperBrowser.Sleep != 0 {
		// Check if wrapper has any sleep defined
		sleep = wrapperBrowser.Sleep
	} else if jobParams.Sleep != 0 {
		// Sleep time in job params takes precedence over anything
		sleep = jobParams.Sleep
	}
	return
}

// Read pools information from job pararams or wrapper browser
func GetProxyPools(jobParams *ctypes.CrawlJobParams, wrapperBrowser ctypes.WrapperBrowser) (pools []string) {

	if len(jobParams.Pools) > 0 {
		pools = jobParams.Pools
	} else if len(wrapperBrowser.Pools) > 0 {
		pools = wrapperBrowser.Pools
	}
	return
}

func GetRequestHeaders(config *types.RequestConfig, wrapperBrowser ctypes.WrapperBrowser) (requestHeaders map[string]string) {
	requestHeaders = make(map[string]string)

	if wrapperBrowser.RequestHeaders != nil {
		requestHeaders = wrapperBrowser.RequestHeaders
	}

	if wrapperBrowser.UserAgent != "" {
		requestHeaders["User-Agent"] = wrapperBrowser.UserAgent
	}
	if wrapperBrowser.Referer != "" {
		requestHeaders["Referer"] = wrapperBrowser.Referer
	}

	if config != nil && config.IsAjax == true && config.ParentUrl != "" {
		requestHeaders["Referer"] = config.ParentUrl
	}

	// Always give precedence to config
	// Preprocess script might send custom headers for ajax requests
	if config != nil && config.Headers != nil {
		for k, v := range config.Headers {
			requestHeaders[k] = v
		}
	}

	return
}

func GetPageTransforms(wrapper *ctypes.Wrapper) (pageTransforms []string) {

	var transforms []string
	wrapperContent := wrapper.Content
	for _, content := range wrapperContent {
		if content.Name == "products" {
			transforms = content.Transform
		}
	}
	if len(transforms) > 0 {
		for _, transform := range transforms {
			pageTransforms = append(pageTransforms, b64.StdEncoding.EncodeToString([]byte(transform)))
		}
	}
	return
}

func ConstructRequestConfig(url string, site string, isAjax bool, workflow *types.CrawlWorkflow) (config types.RequestConfig) {
	if workflow.RdstoreData != nil {
		if workflow.RdstoreData.Crumb != nil {
			config.Crumb = *workflow.RdstoreData.Crumb
		}
	}
	config.ProductMetrics = workflow.ProductMetrics
	config.DomainInfo = workflow.DomainInfo
	config.IsAjax = isAjax
	config.JobType = workflow.ProductMetrics.JobType

	return
}

func Md5Hash(input string) (string, error) {
	hash := md5.New()
	_, err := io.WriteString(hash, input)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func CollectScreenshots(workflow *types.CrawlWorkflow, webResponse types.WebResponse) {
	if workflow.WebResponse.ScreenshotPath == nil {
		workflow.WebResponse.ScreenshotPath = make([]string, 0)
	}

	screenshotPaths := workflow.WebResponse.ScreenshotPath
	for _, screenshotPath := range webResponse.ScreenshotPath {
		screenshotPaths = append(screenshotPaths, screenshotPath)
	}
	workflow.WebResponse.ScreenshotPath = cutils.StringUnique(screenshotPaths)
}

func randomNumber(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

func PrintResponseDetails(status int, details string) {

	var c *color.Color
	switch true {
	case htmlutils.IsTempError(status):
		c = color.New(color.FgHiRed)
	case htmlutils.IsPermError(status):
		c = color.New(color.FgYellow)
	case htmlutils.IsSuccess(status):
		c = color.New(color.FgHiGreen)
	default:
		c = color.New(color.FgHiBlue)
	}

	formatLog := c.SprintFunc()
	log.Println(formatLog(details))
}

func GetContentLength(content string) (int, error) {
	var length int
	var err error
	re, _ := regexp.Compile(`CACHE_WRITTEN: (\d+) bytes`)
	if re.MatchString(content) {
		matches := re.FindStringSubmatch(content)
		if length, err = strconv.Atoi(matches[1]); err != nil {
			return 0, fmt.Errorf("Capturing content size failed with error %v\n", err)
		}
	} else {
		length = len(content)
	}
	return length, nil
}
