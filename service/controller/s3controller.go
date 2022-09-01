package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	"github.com/labstack/echo"
)

func UploadContentToS3(appC *types.Config) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		type ExtractionRequest struct {
			URL         string `json:"url"`
			Content     string `json:"html"`
			JobType     string `json:"job_type"`
			WrapperID   string `json:"wrapper_id"`
			CacheFolder string `json:"cache_folder"`
		}

		var request ExtractionRequest
		if err = c.Bind(&request); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": fmt.Sprintf("Error in binding request: %s\n", err.Error()),
			})
		}

		if request.URL == "" || request.Content == "" || request.JobType == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "url, html and job_type are mandatory fields in the request",
			})
		}

		// Prepare webresponse object
		url := request.URL
		webResponse := &ctypes.WebResponse{
			URL:         url,
			RedirectURL: url,
			Content:     request.Content,
			Success:     true,
			StatusCode:  200,
		}

		var timeTaken float64 = 0.0
		webResponse.TimeTaken = timeTaken
		webResponse.Time = time.Now().Unix()

		// Read job_params from request
		jobType := request.JobType
		jobParams := &ctypes.CrawlJobParams{}
		if request.WrapperID != "" {
			jobParams.WrapperID = request.WrapperID
		}
		if request.CacheFolder != "" {
			jobParams.CacheFolder = request.CacheFolder
		}

		domainInfo, err := utils.GetCompleteDomainInfo(request.URL, request.JobType, appC.ConfigData.WrapperServiceURI, jobParams)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"message": "Could not fetch domain_info for input request",
			})
		}

		var cacheKey string
		if domainInfo != nil {
			var workflow types.CrawlWorkflow
			workflow.DomainInfo = domainInfo
			workflow.JobParams = jobParams
			workflow.RequestId = ""

			// Construct cache_id
			site := domainInfo.DomainName
			cacheID, err := utils.ConstructCacheId(domainInfo.CanonicalUrl, site, jobType, &workflow, domainInfo.Wrapper.Setup.Browser)
			if err != nil {

			}
			domainKey := strings.Replace(site, ".", "_", -1)

			// Check if any cache folder has been sent in job_params
			cacheFolder := "ce"
			if workflow.JobParams.CacheFolder != "" {
				cacheFolder = workflow.JobParams.CacheFolder
			}
			cacheKey = fmt.Sprintf("%s/%s/%s/%s", cacheFolder, jobType, domainKey, cacheID)
			var expiry int32
			expiry = 60 * 60 // seconds (1 hour)
			uploadErr := utils.WriteDataToCache(domainInfo.CanonicalUrl, appC.ConfigData.CacheService, cacheKey, webResponse, expiry)
			if uploadErr != nil {
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"message": fmt.Sprintf("Uploading content to S3 failed with error %s", err.Error()),
				})
			}
		} else {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"message": fmt.Sprintf("Unable to construct domain info object for the request: %v", request),
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": fmt.Sprintf("Successfully uploaded data at %s", cacheKey),
		})
	}
}
