package data

import (
	"fmt"
	"log"
	"strings"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	"github.com/Semantics3/sem3-go-crawl-utils/html"
	rdutils "github.com/Semantics3/sem3-go-crawl-utils/rdstore"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

// RecrawlActions will perform following operations
// Constructing ETL messages from products extracted
// Pusblishing messages to respective queues
func RecrawlActions(url string, workflow *types.CrawlWorkflow, appC *types.Config) (code string, err error) {
	var rawEtlMsgs []*ctypes.RawETLMsg
	// 1. Transform data for recrawl
	rawEtlMsgs, rdstoreUpdateRequest, code, err := PrepareDataForRecrawlETL(workflow)
	if err != nil {
		return code, err
	}
	// 2. Write to rdstore
	err = writeDataToRdstore(workflow.WebResponse.Status, appC.ConfigData.RestRdstoreUpdate, rdstoreUpdateRequest)
	if err != nil {
		return "RDSTORE_WRITE_FAILED", err
	}

	// 4. Pushes to ETL pipeline
	err = publishMsgsToETL(workflow, rawEtlMsgs, appC)
	if err != nil {
		return "ETL_PUBLISH_FAILED", err
	}

	return "", nil
}

// PrepareDataForRecrawlETL will prepare product data for next stages of recrawl ETL pipeline
// Construct message payloads for
// 1. Raw db data consumer
// 2. Processing pipeline data consumer
func PrepareDataForRecrawlETL(workflow *types.CrawlWorkflow) (rawEtlMsgs []*ctypes.RawETLMsg, rdstoreUpdateRequest *ctypes.RdstoreUpdateRequest, code string, err error) {
	url := workflow.URL
	siteName := workflow.DomainInfo.DomainName
	parentSku := workflow.DomainInfo.ParentSku
	rdstoreData := workflow.RdstoreData
	webResponseStatus := workflow.WebResponse.Status
	wrapper := workflow.DomainInfo.Wrapper
	jobInput := workflow.JobInput
	jobParams := jobInput.JobParams
	sitedetail := workflow.DomainInfo.Sitedetail
	vnsp := utils.GetVnspFromWrapper(&wrapper)

	// During recrawl rdstore data MUST be present, else mark it as failed in job-server
	if rdstoreData == nil || rdstoreData.URL == "" {
		return nil, nil, "RDSTORE_DATA_MISSING", fmt.Errorf("RDSTORE_DATA_MISSING for recrawl")
	}

	// SKUS_ONLY variation identification

	// 1. Check if whole site is skus_only from sitedetails
	var skusOnly bool
	if sitedetail.ApiSiteStatus != nil && *sitedetail.ApiSiteStatus == "SKUS_ONLY" {
		skusOnly = true
	}

	// 2. Since late 2018, new products from already indexed site
	// can be indexed as skus_only products
	// eg: amazon.com and walmart.com
	for _, variation := range rdstoreData.Variations {
		if variation.SkusOnly {
			skusOnly = true
			log.Printf("RECRAWL_ACTIONS: Rdstore data for %s has skus_only flag set as %t\n", variation.ChildSku, variation.SkusOnly)
		}
	}

	// 3. While running recrawl jobs for multiple high variation sites in parallel,
	// there's a high chance that our pp consumers in the ETL pipeline might not cope up with speed
	// And at times it could lead to queue explosion and thus leading to pause the whole recrawl
	// To avoid such scenarions where updating Elasticsearch is not a strict requirement,
	// we can set the following flag so that only rawdb gets updates and not match & merged db

	// NOTE: This might lead to inconsistencies b/w 2 dbs
	// Alternatively we can comeup with a slow processing mechanism (like a delayed job may be)
	// which will eventually update the data and thus consistency is maintained

	skusOnlyParam, ok := cutils.GetIntKey(jobParams, "update_skus_only")
	if ok && skusOnlyParam == 1 && !skusOnly {
		skusOnly = true
		log.Printf("RECRAWL_ACTIONS: Found skus_only flag in job_params for job: %s\n", jobInput.JobID)
	}

	if html.IsTempError(webResponseStatus) {
		log.Printf("DATA_TEMPERR: (%s) HTTP %d url, skipping prod data transformations\n", url, webResponseStatus)
		return nil, nil, "", nil
	}

	// Get new and old variations from crawled product variations
	newVariations, oldVariations := GetNewOldVariations(workflow)
	workflow.Data.Products = oldVariations

	var forcediscover bool
	if len(newVariations) > 0 {
		forcediscover = true
	}

	rdstoreUpdateRequest = &ctypes.RdstoreUpdateRequest{
		Site:           siteName,
		ParentSku:      parentSku,
		URL:            url,
		CrawlUpdatedAt: workflow.CrawlTime,
		Vnsp:           vnsp,
		Forcediscover:  forcediscover,
		SkusOnly:       skusOnly,
	}

	if workflow.JobParams.RecrawlFrequency != "" {
		rdstoreUpdateRequest.RecrawlFrequency = workflow.JobParams.RecrawlFrequency
	} else {
		rdstoreUpdateRequest.RecrawlFrequency = "RF3"
	}
	rdstoreVariations := make([]*ctypes.RdstoreChildSKU, 0)

	// Recrawl specific data transformations for processing pipeline consumer ETL stage
	for _, variation := range workflow.Data.Products {
		childSku, ok := cutils.GetStringKey(variation, "_id")
		if !ok {
			log.Printf("DATA_MISSINGSKU: (%s) No _id present for child variation (%v)\n", url, variation)
			continue
		}

		redirectURL, _ := cutils.GetStringKey(variation, "url")
		initURL, ok := cutils.GetStringKey(variation, "_reserved_init_url")
		if !ok || initURL == "" {
			variation["_reserved_init_url"] = redirectURL
			initURL = redirectURL
		}
		variation["ispureoffer"] = 1

		// Rdstore data updates: Begin
		crumb, ok := cutils.GetStringKey(variation, "crumb")
		if ok && len(crumb) >= 3 {
			rdstoreUpdateRequest.Crumb = &crumb
		}
		ncu, ok := cutils.GetStringKey(variation, "_reserved_init_url")
		if ok && strings.HasPrefix(ncu, "http") {
			rdstoreUpdateRequest.Ncu = &ncu
		}

		// b4, _ := variation["images"]
		newImages, ok := cutils.FilterStringList(variation, "images")
		if ok {
			variation["images"] = newImages
		}
		// log.Printf("DATA_IMAGES: Before filtering %v, after filtering %v\n", b4, variation["images"])

		// Update rdstore url only
		// 1. When update_rdstore_url flag is set and
		// 2. Only for active products
		isActive, ok := cutils.GetIntKey(variation, "is_active")
		if ok && isActive > 0 && workflow.JobParams.UpdateRdstoreURL == 1 {
			log.Printf("DATA_RDSTORE_URL_UPDATE: (%s) Updating URL field in rdstore because of update_rdstore_url flag\n", url)
			rdstoreUpdateRequest.URL = redirectURL
		}

		for _, rVariation := range rdstoreData.Variations {
			if rVariation.ChildSku == childSku {
				rdstoreChildSku := &ctypes.RdstoreChildSKU{}
				rdstoreChildSku.ChildSku = rVariation.ChildSku
				rdstoreChildSku.CrawlUpdatedAt = workflow.CrawlTime

				// Add IsActive flag
				if ok && isActive >= 0 {
					rdstoreChildSku.IsActive = true
				} else {
					rdstoreChildSku.IsActive = false
				}

				// Add offers count
				if val, ok := variation["offers"]; ok {
					offers := val.([]interface{})
					rdstoreChildSku.RecentoffersCount = len(offers)
				} else {
					rdstoreChildSku.RecentoffersCount = 0
				}
				rdstoreVariations = append(rdstoreVariations, rdstoreChildSku)
			}
		}
		rdstoreUpdateRequest.Variations = rdstoreVariations
		// Rdstore data updates: End

		// Construct rd-raw-prod message: Begin
		var rawEtlMsg ctypes.RawETLMsg
		rawEtlMsg.MsgID = fmt.Sprintf("%s;%s;%s", jobInput.JobID, siteName, parentSku)
		rawEtlMsg.Data = variation
		rawEtlMsg.Proxy = 0
		if workflow.DomainInfo != nil {
			wrapperProxy := workflow.DomainInfo.Wrapper.Setup.Browser.Proxy
			if wrapperProxy != nil {
				rawEtlMsg.Proxy = *wrapperProxy
			}
		}
		// RdstoreUpdateRequest will have some value all time
		rawEtlMsg.Frequency = rdstoreUpdateRequest.RecrawlFrequency
		em := workflow.Data.ExtractionDataSource
		if em == "" {
			em = "WRAPPER"
		}
		rawEtlMsg.ExtractionMode = em
		rawEtlMsg.DomainName = siteName
		if isActive == 1 {
			rawEtlMsg.Isactive = true
		} else {
			rawEtlMsg.Isactive = false
		}
		if html.IsPermError(webResponseStatus) {
			rawEtlMsg.PageDiscontinued = true
		} else {
			rawEtlMsg.PageDiscontinued = false
		}
		if fdi, ok := cutils.GetIntKey(jobParams, "force_download_image"); ok && fdi == 1 {
			rawEtlMsg.ForceDownloadImage = true
		} else if sitedetail.ForceDownloadImage != nil && *sitedetail.ForceDownloadImage == 1 {
			rawEtlMsg.ForceDownloadImage = true
		}
		if fdis, ok := cutils.GetIntKey(jobParams, "force_download_image_size"); ok && fdis > 0 {
			rawEtlMsg.ForceDownloadImageSize = &fdis
		} else if sitedetail.ForceDownloadImageSize != nil && *sitedetail.ForceDownloadImageSize > 0 {
			rawEtlMsg.ForceDownloadImageSize = sitedetail.ForceDownloadImageSize
		}
		if mvis, ok := cutils.GetIntKey(jobParams, "min_valid_image_size"); ok && mvis > 0 {
			rawEtlMsg.MinValidImageSize = &mvis
		} else if sitedetail.MinValidImageSize != nil && *sitedetail.MinValidImageSize > 0 {
			rawEtlMsg.MinValidImageSize = sitedetail.MinValidImageSize
		}
		if ahis, ok := cutils.GetIntKey(jobParams, "allow_html_image_source"); ok && ahis == 1 {
			rawEtlMsg.AllowHTMLImageSource = true
		} else if sitedetail.AllowHTMLImageSource != nil && *sitedetail.AllowHTMLImageSource == 1 {
			rawEtlMsg.AllowHTMLImageSource = true
		}

		// Rdstore update related
		rawEtlMsg.Vnsp = vnsp
		rawEtlMsg.SkusOnly = &skusOnly

		// Construct rd-raw-prod message: End

		rawEtlMsgs = append(rawEtlMsgs, &rawEtlMsg)
	}

	return rawEtlMsgs, rdstoreUpdateRequest, "", nil
}

// Publish messages to ETL stages
func publishMsgsToETL(workflow *types.CrawlWorkflow, rawEtlMsgs []*ctypes.RawETLMsg, appC *types.Config) (err error) {
	url := workflow.URL
	rawCounter := 0
	ppCounter := 0
	for _, msg := range rawEtlMsgs {
		// Publish to raw db etl queue
		err = appC.RawEtlPublisher.Publish(url, msg)
		if err != nil {
			log.Printf("ETL_PUBLISH_RAW_ERR: failed to publish for (%s), skipping other variations\n", url)
			return err
		}
		rawCounter++

		// Publish to pp etl queue (only if it's skus_only site)
		if !*msg.SkusOnly {
			err = appC.PpEtlPublisher.Publish(url, msg)
			if err != nil {
				log.Printf("ETL_PUBLISH_PP_ERR: failed to publish for (%s), skipping other variations\n", url)
				return err
			}
			ppCounter++
		}
	}
	log.Printf("ETL_PUBLISH_SUCCESS: (%s) Published %d messages to %s, and %d messages to %s\n", url,
		rawCounter, appC.RawEtlPublisher.QueueName, ppCounter, appC.PpEtlPublisher.QueueName)

	return err
}

func writeDataToRdstore(status int, rdstoreService string, rdstoreUpdateRequest *ctypes.RdstoreUpdateRequest) (err error) {
	if html.IsSuccess(status) {
		err = rdutils.UpdateRdstoreData(rdstoreUpdateRequest.URL, rdstoreUpdateRequest.Site, rdstoreService, rdstoreUpdateRequest)
	} else if html.IsPermError(status) {
		rdstoreDiscontinueRequest := &ctypes.RdstoreDiscontinueRequest{
			Site:           rdstoreUpdateRequest.Site,
			ParentSku:      rdstoreUpdateRequest.ParentSku,
			CrawlUpdatedAt: rdstoreUpdateRequest.CrawlUpdatedAt,
		}
		err = rdutils.MarkProductAsDiscontinued(rdstoreUpdateRequest.URL, rdstoreUpdateRequest.Site, rdstoreService, rdstoreDiscontinueRequest)
	}
	return
}
