package m101

import (
	"fmt"

	"github.com/Semantics3/go-crawler/types"
)

// M101 - implements Source interface and
type M101 struct {
	Name      string
	ErrorCode string
}

// GetName - return name
func (m *M101) GetName() string {
	return m.Name
}

// GetErrorCode - return code of error encountered while processing request
func (m *M101) GetErrorCode() string {
	return m.ErrorCode
}

// Request - Make http request to m101 api and fetch response
func (m *M101) Request(url string, workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (canExtract bool, code string, err error) {

	// set the error code before return (datadog metric tracking)
	defer func(c string) {
		m.ErrorCode = code
	}(code)

	var products []map[string]interface{}
	products, code, err = m101Handle.GetResults(appC, url)
	if err != nil {
		return
	}
	// b, _ := json.MarshalIndent(res, "", "  ")
	// fmt.Println("M101_RESPONSE:", string(b))
	product_sku := workflow.DomainInfo.ParentSku
	if workflow.DomainInfo.ParentSku != "" && len(products) >= 1 {
		products[0]["sku"] = product_sku
	}
	workflow.Data.Products = products
	workflow.Data.Status = 1
	if len(workflow.Data.Products) < 1 {
		workflow.Data.Status = 0
		code = "NO_PRODUCT_FROM_SOURCE"
		workflow.Data.Code = "NO_PRODUCT_FROM_SOURCE"
		workflow.Data.Message = "M101 request resulted in an empty products response"
		return false, code, fmt.Errorf("%s", workflow.Data.Message)
	}
	return true, "", nil
}

// Extract - Returns nothing as we get products data directly
func (m *M101) Extract(url string, workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (code string, err error) {
	return
}

// Normalize - normalize m101 data to standard schema
func (m *M101) Normalize(workflow *types.CrawlWorkflow, appC *types.Config) {
}
