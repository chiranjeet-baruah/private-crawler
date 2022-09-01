package data

import (
	"log"
	"strconv"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

// OnDemandCrawlActions will perform necessary actions on how to handle/forward crawl data
func OnDemandCrawlActions(task string, workflow *types.CrawlWorkflow, appC *types.Config) (code string, err error) {

	// Parse task to get line number and input url
	var lineNumber int
	var inputURL string
	matches, didMatch, _ := utils.FindStringSubmatch(task, `^ln\_(\d+)\;(.*)`, "")
	if didMatch {
		ln, err := strconv.Atoi(matches[1])
		if err != nil {
			return "BAD_INPUT", err
		}
		lineNumber = ln
		inputURL = matches[2]
	} else {
		// Handle the cases where proper urls are sent as tasks
		if utils.IsURL(task) {
			inputURL = task
		}
	}

	var status string
	data := make([]map[string]interface{}, 0)
	if workflow.Status == 1 {
		status = "SUCCESS"
		data = append(data, workflow.Data.Products...)
	} else if workflow.FailureType != nil {
		status = *workflow.FailureType
	}

	if workflow.Data.ExtractionEngine != "" && workflow.Data.ExtractionEngine != "WRAPPER" {
		log.Printf("ONDEMAND_CRAWL_ACTIONS_FAILED: Extraction engine is not WRAPPER for %s\n", workflow.URL)
		status = "EXTRACTION FAILED"
	}

	if len(workflow.Data.Products) == 0 {
		log.Printf("ONDEMAND_CRAWL_ACTIONS_FAILED: No crawled products found for %s\n", workflow.URL)
		status = "EXTRACTION FAILED"
	}

	frequency, _ := cutils.GetIntKey(workflow.JobInput.JobParams, "frequency")
	message := &ctypes.OnDemandCrawlWorkflow{
		Site:       workflow.DomainInfo.DomainName,
		ParentSKU:  workflow.DomainInfo.ParentSku,
		JobID:      workflow.JobInput.JobID,
		BatchID:    workflow.JobParams.RunId,
		Client:     workflow.JobParams.Customer,
		InputURL:   inputURL,
		LineNumber: lineNumber,
		Frequency:  frequency,
		Status:     status,
		Data:       data,
	}

	// Publish to ondemand crawl queue
	err = appC.OnDemandCrawlPublisher.Publish(workflow.URL, message)
	if err != nil {
		log.Printf("ONDEMAND_CRAWL_PUBLISH_RAW_ERR: publish failed to publish for %s with error: %v\n", workflow.URL, err)
		return "", err
	}
	return
}
