package controller

import (
	"net/http"

	"github.com/Semantics3/go-crawler/pipeline"
	"github.com/Semantics3/go-crawler/service/helper"
	"github.com/Semantics3/go-crawler/types"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	"github.com/labstack/echo"
)

// Handle crawl requests
func GetCrawlWorkflowHandler(appC *types.Config) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		var workflow types.CrawlWorkflow
		if err = c.Bind(&workflow); err != nil {
			return err
		}
		pipeline.CrawlURL(&workflow, &pipeline.CrawlPipeline{}, appC)
		return c.JSONPretty(http.StatusOK, workflow, "  ")
	}
}

// Handle crawl requests
func GetCrawlSimpleHandler(appC *types.Config) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		var jobInput ctypes.Batch
		if err = c.Bind(&jobInput); err != nil {
			return err
		}
		urls := []string{}
		for k, _ := range jobInput.Tasks {
			urls = append(urls, k)
		}
		batch := helper.JobBatchFromUrls(urls, jobInput.JobDetails.JobType, "crawlendpoint", jobInput.JobParams)
		if jobInput.DataPipeline != nil {
			batch.DataPipeline = jobInput.DataPipeline
		}
		_, crawlResults, err := helper.CrawlJobBatchExecute(batch, appC, "")
		if err != nil {
			return c.JSONPretty(http.StatusInternalServerError, map[string]interface{}{"error": err.Error(), "status": 0}, "  ")
		} else {
			for _, w := range crawlResults {
				if jobInput.JobDetails.JobType != "testwrapper" {
					w.WebResponse.Content = ""
				}
				if jobInput.JobDetails.JobType == "testwrapper" {
					w.JobInput = nil
					//w.RdstoreData = nil
				}
				if w.DomainInfo != nil {
					w.DomainInfo.Wrapper = ctypes.Wrapper{}
				}
			}
			return c.JSONPretty(http.StatusOK, crawlResults, "  ")
		}
	}
}
