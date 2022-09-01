package controller

import (
	"net/http"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	"github.com/labstack/echo"
)

func GetDomainInfo(appC *types.Config) echo.HandlerFunc {

	type getDomainInfoInput struct {
		URL     string `json:"url"`
		JobType string `json:"job_type,omitempty"`
	}

	return func(c echo.Context) (err error) {
		var domainInfoInput getDomainInfoInput

		if err = c.Bind(&domainInfoInput); err != nil {
			return err
		}

		if domainInfoInput.URL == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "url is a required field",
			})
		}

		domainInfo, err := utils.GetPartialDomainInfo(domainInfoInput.URL, domainInfoInput.JobType, appC.ConfigData.WrapperServiceURI)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"message": err.Error(),
			})
		} else {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"domainName":        domainInfo.DomainName,
				"isProductUrl":      domainInfo.IsProductUrl,
				"parent_sku":        domainInfo.ParentSKU,
				"site_status":       domainInfo.SiteStatus,
				"canonicalized_url": domainInfo.CanonicalUrl,
			})
		}
	}

}
