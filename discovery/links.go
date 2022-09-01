package discovery

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	jobutils "github.com/Semantics3/sem3-go-crawl-utils/jobs"
	rdutils "github.com/Semantics3/sem3-go-crawl-utils/rdstore"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

var discoveryMutex = &sync.RWMutex{}

// PrepareDiscoveryCrawlInput will parse input task and fills missing meta info
func PrepareDiscoveryCrawlInput(task string, jobInput *ctypes.Batch) (url string) {
	// TODO: Add sitemap support for op
	_, parentLinkType, url, err := parseTask(task)
	if err != nil {
		// TODO: Handle failed cases
	}

	if parentLinkType == "" {
		parentLinkType = "category"
	}

	discoveryMutex.RLock()
	taskMeta := jobInput.Tasks[task]
	discoveryMutex.RUnlock()

	// Handle cases of missing ancestor information
	if taskMeta.Ancestor == nil && parentLinkType != "" {
		log.Printf("No ancestor information present for %s in meta, adding %s as ancestor", task, parentLinkType)
		taskMeta.Ancestor = &parentLinkType
	}

	// Add default value during missing parent
	if taskMeta.Parent == nil {
		taskMeta.Parent = &parentLinkType
	}

	discoveryMutex.Lock()
	jobInput.Tasks[task] = taskMeta
	discoveryMutex.Unlock()

	return
}

// FilterJobServerFeedbackLinks will filter the wrapper extracted (product) links
// before adding to jobserver feedback. For product urls, it performs rdstore lookup
// to verify if extracted link is eligible for feedback
func FilterJobServerFeedbackLinks(task string, workflow *types.CrawlWorkflow, appC *types.Config) {

	jobID := workflow.JobInput.JobID
	site := workflow.DomainInfo.DomainName

	skippedOutputLinksCount := make(map[string]int)
	groupedOutputLinks := make(map[string][]map[string]string)
	feedbackLinks := make(map[string]ctypes.UrlMetadata)

	// 1. Parse input
	wrapperExtractedLinks := workflow.Data.Links

	// We only care about Category pages
	// On some sites, we extract product links from Product pages as well. Hence, the need to check for extracted links.
	if workflow.DomainInfo.IsProductUrl && len(wrapperExtractedLinks) == 0 {
		return
	}

	outputURLs := []string{}
	for u := range wrapperExtractedLinks {
		outputURLs = append(outputURLs, u)
	}

	// 2. Categorize and group output links as products, categories and sitemap links
	for _, outputURL := range outputURLs {
		outputURLType, op, err := classifyURL(outputURL, task, workflow)
		if err != nil {
			log.Println(err)
			continue
		}
		if !strings.Contains(op, "SKIP_") {
			outputLink := map[string]string{"url": outputURL, "op": op}
			groupedOutputLinks[outputURLType] = append(groupedOutputLinks[outputURLType], outputLink)
		} else {
			skippedOutputLinksCount[op]++
		}
	}

	// 3. Push urls to job server feeback
	totalProductLinksCount := len(groupedOutputLinks["product"])
	if workflow.ProductMetrics.JobType != "testwrapper" {
		// Filter out new products by rdstore lookup
		if totalProductLinksCount > 0 && workflow.JobParams.ForceDiscover == 0 {
			groupedOutputLinks["product"] = filterProductLinks(task, groupedOutputLinks["product"], workflow, appC)
		}

		// Add all the links to jobserver feedback
		for linkType := range groupedOutputLinks {
			addLinksToJobserverFeedback(groupedOutputLinks[linkType], wrapperExtractedLinks, feedbackLinks)
		}
	} else {
		log.Println("Identified jobtype as testwrapper, Skipping queuing items to jobserver feedback")
	}

	// 4. Print spidering output
	totalSkippedLinks := 0
	msg := fmt.Sprintf("TASK: %s, NUM_OUTGOING_LINKS: %d, CATEGORY: %d, SITEMAP: %d, PRODUCTS (TOTAL): %d, PRODUCTS (FILTERED): %d", task, len(outputURLs), len(groupedOutputLinks["category"]), len(groupedOutputLinks["sitemap"]), totalProductLinksCount, len(groupedOutputLinks["product"]))
	for k, v := range skippedOutputLinksCount {
		msg = fmt.Sprintf("%s %s: %d", msg, k, v)
		totalSkippedLinks = totalSkippedLinks + v
	}
	log.Println("SPIDERING_OUTPUT: ", msg)

	// 5. Store spidering output to database
	if workflow.JobParams.SaveSpideringHistory == 1 {
		log.Println("Saving spidering history for ", task)
		spideringOutput := &types.SpideringOutput{
			CreatedAt:            time.Now().Unix(),
			Site:                 site,
			JobID:                workflow.JobInput.JobID,
			ParentLink:           task,
			TotalLinks:           len(outputURLs),
			CategoryLinks:        len(groupedOutputLinks["category"]),
			SitemapLinks:         len(groupedOutputLinks["sitemap"]),
			ProductLinks:         totalProductLinksCount,
			ProductLinksFiltered: len(groupedOutputLinks["product"]),
			SkippedLinks:         totalSkippedLinks,
		}
		err := saveSpideringHistory(spideringOutput, appC)
		if err != nil {
			log.Println(err)
		}
	}

	// 6. For sitemap crawling, upload the tasks directly to jobserver
	if workflow.ProductMetrics.JobType != "testwrapper" && utils.IsSitemapURL(task) && len(feedbackLinks) > 200 {
		log.Printf("JOB_ID: %s, SITE: %s FEEDBACK_LINKS_COUNT: %d. Loading tasks to jobserver directly\n", jobID, site, len(feedbackLinks))
		LoadTasksToJobServer(jobID, feedbackLinks, workflow, appC)
	} else {
		workflow.Data.Links = feedbackLinks
	}
	return
}

func classifyURL(outputURL string, task string, workflow *types.CrawlWorkflow) (outputURLType string, outputURLOp string, err error) {

	jobParams := workflow.JobParams
	site := workflow.DomainInfo.DomainName
	sitedetail := workflow.DomainInfo.Sitedetail

	// 1. Check if outputURL a valid URL
	isURL := utils.IsURL(outputURL)
	if !isURL {
		err = cutils.PrintErr("BAD_URL_EXTRACTED", fmt.Sprintf("Parent:%s, Site: %s, Output link: %s", task, site, outputURL), "")
		return "", "", err
	}

	// 2. Check if outputURL is a sitemapURL
	if utils.IsSitemapURL(outputURL) {
		return "sitemap", "sitemap", nil
	}

	links := workflow.Data.Links
	outputURLMeta := links[outputURL]
	lt := outputURLMeta.LinkType

	// 3. Determine the page type of output url
	isProduct := utils.IsProductURL(outputURL, site, workflow.DomainInfo.Sitedetail)
	if isProduct && (lt == "" || cutils.StringInSlice(lt, []string{"content", "product"})) {
		outputURLType = "product"
		if jobParams.Extraction != "" {
			outputURLOp = jobParams.Extraction
		} else if sitedetail.Extraction != nil && *sitedetail.Extraction == "api" {
			outputURLOp = "api"
		} else {
			outputURLOp = "crawl"
		}
	} else {
		outputURLType = "category"
		outputURLOp = "crawl"
	}

	// Skip rules for category pages
	// 4.1 Skip category page crawl if specified in job_params
	if outputURLType == "category" && jobParams.DontCrawlCategories == 1 {
		outputURLOp = "SKIP_DONT_CRAWL_CATEGORY"
	}

	// 4.2 Skip category page if its extracted from sitemap page
	if utils.IsSitemapURL(task) && outputURLType == "category" {
		outputURLOp = "SKIP_CATEGORY_PAGE_FROM_SITEMAP"
	}

	return outputURLType, outputURLOp, nil
}

func filterProductLinks(task string, outputLinksData []map[string]string, workflow *types.CrawlWorkflow, appC *types.Config) (filteredProductLinks []map[string]string) {
	index := 0
	newProdsCount, rediscoveredProdsCount := 0, 0

	failedUrls := make([]string, 0)
	batchProcessingInput := make([]string, 0)
	for _, value := range outputLinksData {
		batchProcessingInput = append(batchProcessingInput, value["url"])
	}

	utils.BatchProcessItems(batchProcessingInput, 25, func(urls []string) (err error) {

		// 1. Construct input for rdstore lookup by breaking down url to site and sku
		skuBatch := make([]*ctypes.RdstoreParentSKURequest, 0)
		for _, url := range urls {
			// Get domain name
			di, err := utils.GetPartialDomainInfo(url, workflow.ProductMetrics.JobType, appC.ConfigData.WrapperServiceURI)
			if err != nil {
				log.Printf("Extracting domain name for %s failed with error %v\n", url, err)
				failedUrls = append(failedUrls, url)
				index++
				continue
			}
			skuData := &ctypes.RdstoreParentSKURequest{
				Site:      di.DomainName,
				ParentSku: di.ParentSKU,
				URL:       url,
			}
			skuBatch = append(skuBatch, skuData)
		}

		// 2. Peform lookup on batch of skus
		rdstoreBatchResponse, err := rdstoreBatchLookup(skuBatch, workflow, appC)
		if err != nil {
			log.Println(err)
			return err
		}

		for _, skuData := range skuBatch {
			key := fmt.Sprintf("%s;%s", skuData.Site, skuData.ParentSku)
			val, ok := rdstoreBatchResponse[key]
			if ok {
				if val["is_discontinued"].(bool) {
					// Handle cases of discontinued products coming back live
					log.Printf("Discontined product found, adding %s to crawl queue", skuData.URL)
					rediscoveredProdsCount++
					filteredProductLinks = append(filteredProductLinks, outputLinksData[index])
				} else if val["discoverable"].(bool) {
					// Handle new products
					newProdsCount++
					filteredProductLinks = append(filteredProductLinks, outputLinksData[index])
				}
			} else {
				// Handle mis case: rdstore response doesn't have sku info
				newProdsCount++
				filteredProductLinks = append(filteredProductLinks, outputLinksData[index])
			}
			index++
		}
		return nil
	})

	if len(failedUrls) > 0 {
		log.Println("RDSTORE_BULK_LOOKUP_FAILED_COUNT: ", len(failedUrls))
		utils.PrettyJSON("RDSTORE_BULK_LOOKUP_FAILED_URLS", failedUrls, true)
	}

	log.Printf("RDSTORE_BULK_LOOKUP_COMPLETED: New products: %d, Rediscovered products: %d\n", newProdsCount, rediscoveredProdsCount)
	return
}

func addLinksToJobserverFeedback(feedbackData []map[string]string, links map[string]ctypes.UrlMetadata, feedbackLinks map[string]ctypes.UrlMetadata) {
	if len(feedbackData) == 0 {
		return
	}
	for _, v := range feedbackData {
		url := v["url"]
		feedbackTask := fmt.Sprintf("%s;%s", v["op"], url)
		feedbackLinks[feedbackTask] = links[url]
	}
	return
}

// ParseTask will parse a discovery task into op, page tag and URL components
func parseTask(task string) (op string, linkType string, url string, err error) {

	// 1. Manually tagged urls like bestsellers, newreleases and other featured pages
	matches, didMatch, _ := utils.FindStringSubmatch(task, `^(crawl)\_(\w+)\;(.*)`, "")
	if didMatch {
		op = matches[1]
		linkType = matches[2]
		url = matches[3]
		return
	}

	// 2. Regular crawl tasks with crawl prefix (discovery launcher)
	matches, didMatch, _ = utils.FindStringSubmatch(task, `^(?:(api|crawl|sitemap))\;(.*)`, "")
	if didMatch {
		op = matches[1]
		url = matches[2]
		return
	}

	// 3. Plain urls without any prefix (not likely, just handling the case)
	matches, didMatch, _ = utils.FindStringSubmatch(task, `^(https?\:\/\/.*)`, "")
	if didMatch && utils.IsURL(task) {
		op = "crawl"
		url = matches[1]
		return
	}

	if !utils.IsURL(task) {
		err = cutils.PrintErr("BAD_INPUT", fmt.Sprintf("Task: %s is not a url, Skipping", task), "")
	}
	return
}

func saveSpideringHistory(reqBody *types.SpideringOutput, appC *types.Config) (err error) {
	log.Printf("SPIDERING_DATA: (%s) Writing spidering data to database\n", reqBody.Site)
	reqURL := fmt.Sprintf("http://%s/site/spidering", appC.ConfigData.SitesDB)
	user := "discovery-bot"

	// Perform rdstore batch lookup
	body, err := jobutils.RequestUrl("POST", reqURL, reqBody, user)
	if err != nil {
		err = cutils.PrintErr("SPIDERING_REQERR", fmt.Sprintf("failed to %s %s (%v)", "POST", reqURL, reqBody), err)
		return err
	}
	response := make(map[string]interface{})
	err = json.Unmarshal(body, &response)
	if err != nil {
		err = cutils.PrintErr("SPIDERING_DECODEERR", fmt.Sprintf("failed to decode json %s (reqBody: %v) (resp %s)", reqURL, reqBody, string(body)), err)
		return err
	}

	if response["error"] != "" {
		utils.PrettyJSON("SPIDERING_DATA", reqBody, true)
		utils.PrettyJSON("SPIDERING_DATA_RESPONSE", response, true)
	}
	return nil
}

func rdstoreBatchLookup(skuBatch []*ctypes.RdstoreParentSKURequest, workflow *types.CrawlWorkflow, appC *types.Config) (rdstoreResponse ctypes.RdstoreParentSKUBatchResponse, err error) {
	site := workflow.DomainInfo.DomainName
	if len(skuBatch) > 0 {
		rdstoreResponse, err = rdutils.CheckParentSKUBatch(site, appC.ConfigData.RestRdstoreUpdate, skuBatch)
		if err != nil {
			return nil, err
		}
	} else {
		err = cutils.PrintErr("EMPTY_SKU_LIST", fmt.Sprintf("Skus list empty for the batch, Not performing rdstore lookup. Batch(%v\n)", skuBatch), "")
		return nil, err
	}

	return rdstoreResponse, nil
}
