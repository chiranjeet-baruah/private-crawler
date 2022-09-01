package utils

import (
	"fmt"
	"log"
	"strings"

	"github.com/Semantics3/go-crawler/types"
	sitedetails "github.com/Semantics3/sem3-go-crawl-utils/sitedetails"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

// AssignRequestId will assign a unique request_id for each request (based on jobType)
func AssignRequestId(jobType string, workflow *types.CrawlWorkflow) {
	if jobType == "recrawl" || (jobType == "realtimeapi" && workflow.JobParams.Cache != 1) {
		if workflow.JobParams.RequestId != "" {
			workflow.RequestId = workflow.JobParams.RequestId
		} else {
			workflow.RequestId = GenerateUniqueId(32)
		}
	}
}

func CheckIfRedirectSkuChange(workflow *types.CrawlWorkflow) (err error) {

	parentSku := workflow.DomainInfo.ParentSku

	// Canonicalize redirect URL
	redirectUrl := workflow.WebResponse.Redirect
	cRedirectUrl, err := sitedetails.CanonicalizeUrl(redirectUrl, workflow.DomainInfo.Sitedetail)
	if err != nil {
		err = cutils.PrintErr("CRAWL_REDIRECT_CANONICALIZEERR", fmt.Sprintf("failed to canonicalize redirect url: %s", redirectUrl), err)
		return err
	}

	// Extract parent sku from redirect URL
	redirectSku, match, err := sitedetails.ApplySitedetailSkuRegex(cRedirectUrl, workflow.DomainInfo.Sitedetail)
	if match != true || err != nil {
		err = cutils.PrintErr("CRAWL_REDIRECT_SKUREGEXERR", fmt.Sprintf("failed to extract sku from redirect url: %s", cRedirectUrl), err)
		return err
	}

	if redirectSku == "" || (parentSku != redirectSku) {
		err = cutils.PrintErr("CRAWL_REDIRECT_SKUMISMATCHERROR", fmt.Sprintf("Redirect sku %s doesn't match parent sku %s", redirectSku, parentSku), "")
		return err
	}

	return
}

// Return failed workflow result for job-server batch
func FailWorkflow(task string, pipeline types.Pipeline, w *types.CrawlWorkflow, ftype string, fmsg string, appC *types.Config) {
	w.Status = 0
	w.FailureType = &ftype
	w.FailureMessage = &fmsg
	domain, parentSku := "", ""
	if w.DomainInfo != nil {
		domain = w.DomainInfo.DomainName
		parentSku = w.DomainInfo.ParentSku
	}
	log.Printf("WORKFLOW_FAILED: (%s ~> %s;%s) %s %s\n", w.URL, domain, parentSku, ftype, fmsg)

	if pipeline.ShouldCallPostCrawlOpsOnFailure(w) && !w.PostCrawlOpsCalled && !w.PreCrawlOpsFailed {
		code, err := pipeline.PostCrawlOps(task, w, appC)
		w.PostCrawlOpsCalled = true
		if err != nil {
			errMsg := err.Error()
			w.FailureType = &code
			w.FailureMessage = &errMsg
		}
	}
}

// Pretty print domain info
func PrintDomainInfo(di ctypes.DomainInfo) {
	di.Wrapper = ctypes.Wrapper{}
	log.Printf("DomainInfo\n")
	PrettyJSON("DOMAIN_INFO", di, true)
}

// Pretty print workflow
func PrintResults(crawlResults map[string]*types.CrawlWorkflow) {
	if crawlResults == nil {
		log.Printf("PRETTYPRINT: empty results sent\n")
		return
	}

	for u, w := range crawlResults {
		errCode := ""
		errMsg := ""
		if w.FailureType != nil && *w.FailureType != "" {
			errCode = *w.FailureType
			errMsg = *w.FailureMessage
		}
		log.Printf("PRINTRESULT_WEBRESP: (%s, %s) (PageContent: %d bytes, HTTP Status: %d, Time Taken: %.2f secs, Redirect: %s, errCode: %s, errMsg: %s)\n",
			u, w.DomainInfo.DomainName, len(w.WebResponse.Content), w.WebResponse.Status, w.WebResponse.TimeTaken, w.WebResponse.Redirect, errCode, errMsg)
		log.Printf("PRINTRESULT_DATAEXTRACTED\n")
		PrettyJSON("DATA", w.Data, true)

		if w.ValidateErrors != nil && (w.ValidateErrors.Errs != nil || w.ValidateErrors.Warn != nil) {
			log.Printf("PRINTVALIDATION_RES\n")
			PrettyJSON("VALIDATE_ERRORS", w.ValidateErrors, true)
		}
	}
}

// Get active prods from content extraction response
func GetActiveProds(url string, workflow *types.CrawlWorkflow) (activeProds int, avgFields float64, totalProds int) {
	if workflow.Data.Products != nil {
		totalProds = len(workflow.Data.Products)
		for _, variation := range workflow.Data.Products {
			roc, ok1 := cutils.GetIntKey(variation, "recentoffers_count")
			isd, ok2 := cutils.GetIntKey(variation, "isdiscontinued")
			if !(ok1 && ok2 && roc == 0 && isd == 1) {
				activeProds++
				avgFields = avgFields + float64(len(variation))
			}
		}
		avgFields = avgFields / float64(activeProds)
	}
	return activeProds, avgFields, totalProds
}

// GetExtractionMode - Construct extraction mode based on the data sources sent in input
func GetExtractionMode(dataSources []string) string {

	// DEPRECATED
	// NOTE: Extraction mode explicitly specified in the sitedetail takes precedence over the
	// extraction mode dynamically set for the job using the `extraction_mode` option in job params
	// if di != nil && di.Sitedetail != nil && di.Sitedetail.ExtractionMode != nil {
	// 	em := *di.Sitedetail.ExtractionMode
	// 	if em != "" {
	// 		return em
	// 	}
	// }
	// if jobParams != nil && jobParams.ExtractionMode != "" {
	// 	return jobParams.ExtractionMode
	// }

	if len(dataSources) > 0 {
		return strings.Join(dataSources, ",")
	}
	return "WRAPPER"
}

// Print crawl summary
func PrintCrawlSummary(url string, workflow *types.CrawlWorkflow) {
	activeProds, avgFields, totalProds := GetActiveProds(url, workflow)
	log.Printf("CRAWL_END: (%s) [Active products: %d, Total: %d, Avg fields: %.2f, Basic: 10]\n", url, activeProds, totalProds, avgFields)
	if avgFields <= 10 {
		PrettyJSON("PRODUCTS", workflow.Data.Products, true)
	}
}

func GetCrawlType(jobType string, jobParams *ctypes.CrawlJobParams) (crawlType string) {
	if jobType == "discovery_crawl" {
		if jobParams.DiscoveryCrawlType != "" {
			crawlType = jobParams.DiscoveryCrawlType
		}
	} else {
		crawlType = "RECRAWL"
	}
	return
}

func GetVnspFromWrapper(wrapper *ctypes.Wrapper) (vnsp bool) {
	val := sitedetails.CheckWrapperForVnsp(wrapper)
	if val == "1" {
		vnsp = true
	} else {
		vnsp = false
	}
	return
}
