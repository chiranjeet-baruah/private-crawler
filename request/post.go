package request

import (
	"bytes"
	"fmt"
	"log"
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

// PostRequest - makes post request using crawler
func PostRequest(url string, config *types.RequestConfig, jobParams *ctypes.CrawlJobParams, appC *types.Config) (webResponse types.WebResponse) {

	webResponse.URL = url
	log.Printf("POST_REQ_START: URL: %s, REQ_BODY: %s\n", url, config.Body)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(config.Body)))
	if err != nil {
		err = cutils.PrintErr("POST_REQ_REQUEST_CREATION_FAILED", fmt.Sprintf("Creating HTTP request for %s failed with error", url), err)
		webResponse.Error = err.Error()
		return
	}

	for k, v := range config.Headers {
		req.Header.Add(k, v)
	}

	timeout := 60
	if config.Timeout != 0 {
		timeout = config.Timeout
	}

	var client *http.Client

	// If cookies are sent in configure them in request
	// Cookies is a string array with items separated by ; as a delimiter

	// Example cookie
	// uid=lo_2arJDgHVfLth; sid=1:kgKYJ1ic8F01gkBRemC5M+nZwCyeCkwtYqYkM+VLpQsQ/suDLuh8edKWuwSQiRpT; optimizelyEndUserId=lo_2arJDgHVfLth; __cfduid=d09d2673ac52f143081d37fc75a711a111593933381; lightstep_guid/lite-web=0ed6453a50db20d6; lightstep_session_id=5bc62a0d1a7865df; __cfruid=c654c31b645dbfe97a384ee46056362d254794a3-1595226702
	if config.Cookie != "" {
		cookies := strings.Split(config.Cookie, ";")
		for _, cookie := range cookies {

			// Each cookie value is string of key/value pair with items separated by = as a delimiter
			data := strings.Split(cookie, "=")
			if len(data) > 1 {
				req.AddCookie(&http.Cookie{Name: data[0], Value: data[1]})
			}
		}
	}

	if client == nil {
		client = &http.Client{
			Timeout: time.Second * time.Duration(timeout),
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		err = handleError(url, err)
		webResponse.Error = err.Error()
		return
	}

	webResponse.Status = resp.StatusCode
	content, err := htmlutils.GetContentFromResponse(resp)
	if err != nil {
		err = cutils.PrintErr("POST_REQ_RESPONSE_PARSE_FAILED", fmt.Sprintf("Parsing response %v for %s failed", resp.Body, url), err)
		webResponse.Error = err.Error()
		return
	}
	webResponse.Content = content
	webResponse.Time = time.Now().Unix()
	if config.CacheKey != "" {
		err := utils.WriteDataToCache(url, appC.ConfigData.CacheService, config.CacheKey, webResponse, config.CacheExpiry)
		log.Println(err)
	}

	return
}

func handleError(url string, err error) error {
	status := http.StatusInternalServerError
	message := err.Error()
	timeout := regexp.MustCompile(`Client.Timeout exceeded while awaiting headers`)
	if timeout.MatchString(message) {
		status = http.StatusRequestTimeout
		message = fmt.Sprintf("Request timed out for %s", url)
	}
	return cutils.PrintErr("POST_REQ_REQUEST_FAILED", fmt.Sprintf("Request for %s failed with Status: %d, Error: %s\n", url, status, message), "")
}
