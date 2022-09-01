package utils

import (
	"encoding/json"
	"fmt"

	"github.com/Semantics3/sem3-go-crawl-utils/jobs"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
)

type DomainNameResp struct {
	DomainName string `json:"domain_name,omitempty"`
	Message    string `json:"message,omitempty"`
}

// GetDomainName - Callers of this function are interested only in partial domain info
// So no need for fetching sitedetail/wrapper information

func GetDomainName(url, wrapperServiceURI string) (domain string, err error) {
	reqBody := map[string]interface{}{
		"url": url,
	}
	reqURL := fmt.Sprintf("http://%s/domain/url", wrapperServiceURI)
	bodyBytes, err := jobs.RequestUrl("POST", reqURL, reqBody, "proxy-node")
	if err != nil {
		return domain, fmt.Errorf("DOMAIN_EXTRACT_FETCH_ERR: url: %s, %v", url, err)
	}

	resp := &DomainNameResp{}
	err = json.Unmarshal(bodyBytes, resp)
	if err != nil {
		return domain, fmt.Errorf("DOMAIN_EXTRACT_UNMARSHAL_ERR: url: %s, body %s, Err: %v", url, string(bodyBytes), err)
	}

	/*
	   Response will be like the following based on success/failure cases
	   {
	        "domain_name": "nubianskin.com"

	   }
	   {
	        "message": "error message"

	   }
	*/
	if resp.DomainName != "" {
		domain = resp.DomainName
	} else if resp.Message != "" {
		err = fmt.Errorf("DOMAIN_EXTRACT_ERR: url: %s, error: %s", url, resp.Message)
	} else if resp.Message == "" && resp.DomainName == "" {
		err = fmt.Errorf("DOMAIN_EXTRACT_ERR: empty domain name extracted for %s", url)
	}

	return domain, err
}

// GetDomainInfoWithWrapper - Callers of this function are interested only in complete domain info
func GetCompleteDomainInfo(url, jobType, wrapperServiceURI string, jobParams *ctypes.CrawlJobParams) (di *ctypes.DomainInfo, err error) {
	di = &ctypes.DomainInfo{}
	err = requestWrapperService(url, jobType, wrapperServiceURI, 1, jobParams, di)
	return
}

// GetDomainInfo - Callers of this function are interested only in partial domain info
// So no need for fetching sitedetail/wrapper information
func GetPartialDomainInfo(url, jobType, wrapperServiceURI string) (di *ctypes.DomainInfoCompact, err error) {
	di = &ctypes.DomainInfoCompact{}
	err = requestWrapperService(url, jobType, wrapperServiceURI, 0, nil, di)
	return
}

// Retrieve domain info from wrapper-service
// Following endpoint is used by various clients for different use cases
// So sending sitedetail/wrapper in the response is optional to avoid unnecessary network transfer
// As crawler needs that info, it has to ask for it explicitly
func requestWrapperService(url, jobType, wrapperServiceURI string, sendWrapper int, jobParams *ctypes.CrawlJobParams, resp interface{}) error {
	reqBody := map[string]interface{}{
		"url":          url,
		"job_type":     jobType,
		"job_params":   jobParams,
		"send_wrapper": sendWrapper,
	}
	reqURL := fmt.Sprintf("http://%s/domain/info", wrapperServiceURI)
	bodyBytes, err := jobs.RequestUrl("POST", reqURL, reqBody, fmt.Sprintf("%s-crawler", jobType))
	if err != nil {
		return fmt.Errorf("FETCH_ERR: %v", err)
	}

	err = json.Unmarshal(bodyBytes, resp)
	if err != nil {
		return fmt.Errorf("UNMARSHAL_ERR: Body %s, Err: %v", string(bodyBytes), err)
	}

	return err
}
