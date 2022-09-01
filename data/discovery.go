package data

import (
	"fmt"
	"log"
	"strings"

	"github.com/Semantics3/go-crawler/discovery"
	"github.com/Semantics3/go-crawler/types"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
	"github.com/jinzhu/copier"
)

func DiscoveryActions(url string, workflow *types.CrawlWorkflow, appC *types.Config) (code string, err error) {

	if len(workflow.Data.Products) > 0 {

		// 1. If forcediscover is not set, send old variations through recrawl pipeline
		if workflow.DomainInfo.IsProductUrl && workflow.JobParams.ForceDiscover == 0 {
			sendOldVariatonsToRecrawl(url, workflow, appC)
		}

		// 2. Write product data to mongo
		// TODO: Add support for workflow.Data.Categories later
		// NOTE: Removing IsProductURL and non-empty ParentSKU checks as
		// 2.1 Crawl collections are not created for search pages
		// 2.2 As rdstore writes are offloaded to skus workers, no need for strict checks here
		if appC.MongoCrawl == nil {
			err = cutils.PrintErr("MONGO_CONNECTION_MISSING", fmt.Sprintf("Failed to find mongo crawl connection object for %s", workflow.JobInput.JobID), err)
			return "MONGO_CONNECTION_MISSING", err
		}
		err = discovery.WriteCrawlDataToMongo(workflow, appC)
		if err != nil {
			return "MONGO_WRITE_FAILED", err
		}
	}
	return "", nil
}

// Decouple old variations from products and send them to recrawl pipeline
// If forcediscover flag is present or if its a discontinued product came back live let all the variations proceed
func sendOldVariatonsToRecrawl(url string, workflow *types.CrawlWorkflow, appC *types.Config) {
	newVariations, oldVariations := GetNewOldVariations(workflow)
	if len(oldVariations) > 0 {

		// Construct recrawl workflow for old variations and use recrawl as job_type since
		// rawdb-data consumers rely on job_id being recrawl type to upsert or update data
		// in skus db
		recrawlWorkflow := types.CrawlWorkflow{}
		copier.Copy(&recrawlWorkflow, workflow)

		// Send all the old variations through recrawl pipeline
		go func(u string, w *types.CrawlWorkflow, old []map[string]interface{}, a *types.Config) {

			jobInput := ctypes.Batch{}
			copier.Copy(&jobInput, w.JobInput)
			jobInput.JobID = strings.Replace(jobInput.JobID, "discovery_crawl", "recrawl", -1)
			jobInput.JobDetails.JobType = "recrawl"
			jobInput.JobDetails.JobID = jobInput.JobID
			jobInput.JobParams["frequency"] = "discovery_crawl"
			w.JobInput = &jobInput

			log.Printf("RECRAWL_PIPELINE_SEND: (%s) Sending %d old variations from %s to recrawl pipeline", recrawlWorkflow.JobInput.JobID, len(old), u)
			w.Data.Products = old
			code, err := RecrawlActions(u, w, a)
			if err != nil {
				log.Printf("Processing old variations for %s failed with error %s: %v\n", u, code, err)
			}
		}(url, &recrawlWorkflow, oldVariations, appC)
	}
	workflow.Data.Products = newVariations
	log.Printf("Sending %d new variations from %s to discovery pipeline", len(workflow.Data.Products), url)
}
