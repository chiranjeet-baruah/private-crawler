package amazon

import (
	"fmt"

	"github.com/Semantics3/go-crawler/types"
)

// Amazon - implements Source interface and
type Amazon struct {
	Name      string
	ErrorCode string
}

// GetName - return name
func (a *Amazon) GetName() string {
	return a.Name
}

// GetErrorCode - return code of error encountered while processing request
func (a *Amazon) GetErrorCode() string {
	return a.ErrorCode
}

// Request - Make http request to Amazon api and fetch response
func (a *Amazon) Request(url string, workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (canExtract bool, code string, err error) {

	// set the error code before return (datadog metric tracking)
	defer func(c string) {
		a.ErrorCode = code
	}(code)

	var products []map[string]interface{}
	jobType := workflow.JobInput.JobDetails.JobType
	if jobType == "realtimeapi" {
		products, code, err = amazonHandle.GetVariations(appC, url, jobType)

		// Validate if queried ASIN is present in resp
		valid_resp := 0
		queried_asin := workflow.DomainInfo.ParentSku
		if workflow.DomainInfo.ParentSku != "" && len(products) >= 1 {
			for _, item := range products {
				resp_asin := item["sku"]
				if queried_asin == resp_asin {
					valid_resp = 1
				}
			}
			// Else get results and append to top of array
			if valid_resp == 0 {
				var product []map[string]interface{}
				product, code, err = amazonHandle.GetItems(appC, url, jobType)
				// As the missing ASIN is same variant
				// copy over the variation_id
				product[0]["variation_id"] = products[0]["variation_id"]
				// Make queried ASIN first element of products array
				temp := products[0]
				copy(products[0:], product[0:])
				products = append(products, temp)
			}
		}

		// If there are no variations
		// Get single item only
		if err != nil {
			products, code, err = amazonHandle.GetItems(appC, url, jobType)
		}
	} else {
		products, code, err = amazonHandle.GetItems(appC, url, jobType)
	}
	if err != nil {
		return
	}

	workflow.Data.Products = products
	workflow.Data.Status = 1
	if len(workflow.Data.Products) < 1 {
		workflow.Data.Status = 0
		code = "AMAZON_NO_PRODUCTS_ERR"
		workflow.Data.Code = code
		workflow.Data.Message = "AMAZON PAAPI request resulted in an empty products response"
		return false, code, fmt.Errorf("%s", workflow.Data.Message)
	}
	return true, "", nil
}

// Extract - Returns nothing as we get products data directly
func (a *Amazon) Extract(url string, workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (code string, err error) {
	return
}

// Normalize - normalize Amazon data to standard schema
func (a *Amazon) Normalize(workflow *types.CrawlWorkflow, appC *types.Config) {
}
