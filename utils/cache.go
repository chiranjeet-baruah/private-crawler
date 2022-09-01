package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Semantics3/go-crawler/types"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
	htmlCache "github.com/Semantics3/sem3-go-crawl-utils/webcache/html"
	s3Cache "github.com/Semantics3/sem3-go-crawl-utils/webcache/s3"
)

// Marshal struct to a JSON string but ensure that the process is repeatable
func canonicalJsonMarshal(c types.CacheKeyConfig, url string) (string, error) {
	rBytes, _ := json.Marshal(c)
	r := make(map[string]interface{}, 0)
	if err := json.Unmarshal(rBytes, &r); err != nil {
		err = cutils.PrintErr("JSON_UNMARSHAL_FAILED", fmt.Sprintf("Failed to unmarshal cache config object to a map for: %s, %v", url, c), err)
		return "", err
	}
	rSortedKeys, err := json.Marshal(&r)
	if err != nil {
		err = cutils.PrintErr("JSON_MARSHAL_FAILED", fmt.Sprintf("Failed to marshal cache config object for: %s, %s", url, string(rBytes)), err)
		return "", err
	}
	return string(rSortedKeys), nil
}

// Construct a unique ID based on the URL and request config
// Send nil values as request config, as respective fields
// are needed only for ajax requests
func ConstructCacheId(url string, domain string, jobType string, workflow *types.CrawlWorkflow, wrapperBrowser ctypes.WrapperBrowser) (string, error) {
	var c types.CacheKeyConfig
	c.Url = url
	c.Domain = domain
	c.Headers = GetRequestHeaders(nil, wrapperBrowser)
	c.RequestPolicy = GetRequestPolicy(workflow.JobParams, wrapperBrowser)
	c.Cookie = GetCookies(c.RequestPolicy, nil, workflow.JobParams, wrapperBrowser)

	// Get request_id
	if workflow.RequestId != "" {
		c.RequestId = workflow.RequestId
	}

	canonicalJson, err := canonicalJsonMarshal(c, url)
	if err != nil {
		return "", err
	}
	rHash, err := Md5Hash(canonicalJson)
	if err != nil {
		return "", cutils.PrintErr("CREATING_MD5HASH_FAILED", fmt.Sprintf("Failed to create md5hash for: %s", canonicalJson), err)
	}
	log.Printf("CACHE_KEY_FIELDS: %s, CACHE_KEY_GENERATED: %s\n", canonicalJson, string(rHash))
	return string(rHash), nil
}

// ReadDataFromCache Read html data for the request from cache
// Checks if request is eligible for reading from cache based on
// 1. Flag sent through job params
// 2. When was the object written to cache (ttl)
func ReadDataFromCache(cacheServiceHost, cacheKey string, workflow *types.CrawlWorkflow) {
	start := time.Now()
	jobParams := workflow.JobParams

	// Lookup against cache
	bodyBytes, err := htmlCache.DownloadHtmlUsingCacheService(cacheServiceHost, cacheKey)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// If lookup is successful, uncompress and decode to struct
	resp := &ctypes.WebResponse{}
	err = json.Unmarshal(bodyBytes, resp)
	if err != nil {
		log.Printf("CACHE_READ_JSON_DECODE_ERROR: location: %s, error: %v", cacheKey, err)
		return
	}

	// Check if cached data is obsolete
	// Default cache expiry time is 60 mins
	ttl := jobParams.CacheTtl
	if ttl == 0 {
		ttl = 1 * 60 * 60
	}

	if ttl > 0 {
		end := time.Now()
		duration := end.Sub(time.Unix(resp.Time, 0))
		seconds := int(duration.Seconds())
		if seconds >= ttl {
			log.Printf("CACHE_READ_TTL_EXCEEDED_ERROR: location: %s, age: %d seconds", cacheKey, seconds)
			return
		}
	}

	// TestWrapper pipeline downloads cache at the final step
	// So that data can be displayed on console
	// For the initial requests, headers will be empty
	// For 2nd time (final step) headers will not be empty
	var responseHeaders ctypes.ResponseHeaders
	if workflow.WebResponse.Headers.XNodePool != "" {
		responseHeaders = workflow.WebResponse.Headers
	}

	if resp.Content != "" {
		// Construct web response object
		workflow.WebResponse = types.WebResponse{
			URL:       resp.URL,
			Redirect:  resp.RedirectURL,
			Status:    resp.Status,
			Success:   resp.Success,
			Time:      resp.Time,
			Cookie:    resp.Cookie,
			Headers:   resp.Headers,
			Message:   resp.Message,
			Content:   resp.Content,
			FromCache: true,
			Attempts:  1,
			TimeTaken: ComputeDuration(start),
		}
		workflow.CrawlTime = resp.Time

		// Copy back the original response headers from the workflow
		if responseHeaders.XNodePool != "" {
			workflow.WebResponse.Headers = responseHeaders
		}

	} else {
		log.Printf("CRAWL_CACHEMISS: (%s) s3path %s\n", workflow.URL, workflow.CacheKey)
		// FIXME: this is not the right place for this
		workflow.CrawlTime = time.Now().Unix()
	}
}

// WriteDataToCache - uploads data to cache using cache service
func WriteDataToCache(url, cacheService, cacheKey string, data interface{}, expiry int32) error {
	cacheContent, err := json.Marshal(data)
	if err != nil {
		err = cutils.PrintErr("WEBCACHE_JSON_ENCODE_ERROR", fmt.Sprintf("Error encoding cache document: %v\n", data), err)
		return err
	} else {
		_, err = htmlCache.UploadHtmlUsingCacheService(cacheService, cacheKey, cacheContent, expiry)
		if err != nil {
			err = cutils.PrintErr("WEBCACHE_WRITE_ERROR", fmt.Sprintf("Error writing to cache : (%s) %s\n", url, cacheKey), err)
			return err
		}
	}

	return nil
}

// ReadDataFromS3 Read html data for the request from S3
// Checks if request is eligible for reading from cache based on
// 1. Flag sent through job params
// 2. When was the object written to cache (ttl)
func ReadDataFromS3(cacheKey string, workflow *types.CrawlWorkflow, s3Client *s3Cache.Client) {
	start := time.Now()

	// Default cache expiry time is 60 mins
	ttl := workflow.JobParams.CacheTtl
	if ttl == 0 {
		ttl = 1 * 60 * 60
	}

	// Download data from S3
	bodyBytes, err := s3Client.DownloadDataFromS3(cacheKey)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// Decompress data
	decompressedContentBytes, err := htmlCache.DecompressTextContent(bodyBytes)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// If lookup is successful, uncompress and decode to struct
	resp := &ctypes.WebResponse{}
	err = json.Unmarshal(decompressedContentBytes, resp)
	if err != nil {
		log.Printf("CACHE_READ_JSON_DECODE_ERROR: location: %s, error: %v", cacheKey, err)
		return
	}

	// Check if cached data is obsolete
	if ttl > 0 {
		end := time.Now()
		duration := end.Sub(time.Unix(resp.Time, 0))
		seconds := int(duration.Seconds())
		if seconds >= ttl {
			log.Printf("CACHE_READ_TTL_EXCEEDED_ERROR: location: %s, age: %d seconds", cacheKey, seconds)
			return
		}
	}

	if resp.Content != "" {
		// Construct web response object
		workflow.WebResponse = types.WebResponse{
			URL:       resp.URL,
			Redirect:  resp.RedirectURL,
			Status:    resp.Status,
			Success:   resp.Success,
			Time:      resp.Time,
			Cookie:    resp.Cookie,
			Headers:   resp.Headers,
			Message:   resp.Message,
			Content:   fmt.Sprintf("CACHE_READ_FROM_S3: (%s) %s", workflow.URL, cacheKey),
			FromCache: true,
			Attempts:  1,
			TimeTaken: ComputeDuration(start),
		}
		workflow.CrawlTime = resp.Time
	} else {
		log.Printf("CRAWL_CACHEMISS: (%s) s3path %s\n", workflow.URL, workflow.CacheKey)
		// FIXME: this is not the right place for this
		workflow.CrawlTime = time.Now().Unix()
	}
}

// WriteDataToS3 - uploads data to s3
func WriteDataToS3(url, cacheKey string, data interface{}, s3Client *s3Cache.Client) error {
	cacheContent, err := json.Marshal(data)
	if err != nil {
		err = cutils.PrintErr("WEBCACHE_JSON_ENCODE_ERROR", fmt.Sprintf("Error encoding cache document: %v\n", data), err)
		return err
	} else {
		// Compress data
		compressedContentBytes := htmlCache.CompressTextContent(cacheContent)

		// Set expiry for the cache object
		err = s3Client.UploadData(cacheKey, string(compressedContentBytes), false)
		if err != nil {
			err = cutils.PrintErr("WEBCACHE_WRITE_ERROR", fmt.Sprintf("Error writing to cache : (%s) %s\n", url, cacheKey), err)
			return err
		}
	}

	return nil
}
