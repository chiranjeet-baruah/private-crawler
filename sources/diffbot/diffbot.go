package diffbot

import (
	"errors"
	"strings"

	"github.com/Semantics3/go-crawler/sources"
	"github.com/Semantics3/go-crawler/types"
)

// Diffbot - implements Source interface and
type Diffbot struct {
	Name      string
	ErrorCode string
}

// Request - Make http request to m101 api and fetch response
func (d *Diffbot) Request(url string, workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (canExtract bool, code string, err error) {

	// set the error code before return (datadog metric tracking)
	defer func(c string) {
		d.ErrorCode = code
	}(code)

	engine := "DIFFBOT"
	args := []interface{}{workflow}
	err = sources.MakeRPCRequest(appC.RPCClient, engine, url, "extractWithDiffbot", args, &workflow.Data)
	if err != nil {
		if strings.Contains(err.Error(), "Unable to extract") {
			code = "DIFFBOT_EXTRACTION_FAILED"
		}
		return canExtract, code, err
	}

	if workflow.Data.Status == 0 {
		if strings.Contains(workflow.Data.Message, "Unable to extract") {
			code = "DIFFBOT_EXTRACTION_FAILED"
		}
		return canExtract, code, errors.New(workflow.Data.Message)
	}
	return
}

// Extract - Returns nothing as we get products data directly
func (d *Diffbot) Extract(url string, workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (code string, err error) {
	return
}

// Normalize - normalize m101 data to standard schema
func (d *Diffbot) Normalize(workflow *types.CrawlWorkflow, appC *types.Config) {
	return
}

// GetName - return name
func (d *Diffbot) GetName() string {
	return d.Name
}

// GetErrorCode - return code of error encountered while processing request
func (d *Diffbot) GetErrorCode() string {
	return d.ErrorCode
}
