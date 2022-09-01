package controller

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"

	"github.com/Semantics3/go-crawler/request"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
)

// GetScreenshotHandler - screenshot controller
func GetScreenshotHandler(appC *types.Config) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		var req types.ScreenshotRequest
		resp := types.ScreenshotResponse{}
		if err = c.Bind(&req); err != nil {
			resp.FailureType = "REQUEST_PARAM_PARSE_ERROR"
			resp.FailureMessage = fmt.Sprintf("Error in binding request: %s\n", err.Error())
			return c.JSON(http.StatusBadRequest, resp)
		}
		if req.URL == "" {
			resp.FailureType = "REQUIRED_PARAM_EMPTY"
			resp.FailureMessage = "url is a mandatory field in the request"
			return c.JSON(http.StatusBadRequest, resp)
		}
		url := req.URL

		domain, err := utils.GetDomainName(url, appC.ConfigData.WrapperServiceURI)
		if err != nil {
			resp.FailureType = "SITE_EXTRACTION_ERROR"
			resp.FailureMessage = fmt.Sprintf("site name could not be extracted: %s", err.Error())
			return c.JSON(http.StatusBadRequest, resp)
		}
		req.Domain = domain
		if len(req.Pools) == 0 {
			req.Pools = []string{"internal_chrome"}
		}
		if req.Timeout == 0 {
			req.Timeout = 60
		}
		if req.RequestPolicy == "" {
			req.RequestPolicy = "render:1;rendering_engine:render_chrome;render_wait:10;"
		}
		config := &types.RequestConfig{}
		req.RequestPolicy = utils.GetScreenshotPath(req.Domain, url, true, req.RequestPolicy, config, &ctypes.CrawlJobParams{})

		err = request.GetScreenshot(appC, req)
		if err != nil {
			resp.FailureType = "PROXY_CLOUD_REQUEST_ERROR"
			resp.FailureMessage = fmt.Sprintf("Error: %s", err.Error())
			return c.JSON(http.StatusInternalServerError, resp)
		}

		resp.Status = 1
		resp.Screenshot = config.ScreenshotPath
		return c.JSONPretty(http.StatusOK, resp, "  ")
	}
}
