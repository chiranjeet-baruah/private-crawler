package merge

import (
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/Semantics3/go-crawler/sources/amazon"
	"github.com/Semantics3/go-crawler/sources/diffbot"
	"github.com/Semantics3/go-crawler/sources/m101"
	"github.com/Semantics3/go-crawler/sources/supervised"
	"github.com/Semantics3/go-crawler/sources/unsupervised"
	"github.com/Semantics3/go-crawler/stats"
	"github.com/Semantics3/go-crawler/types"
	// cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

type hash = map[string]interface{}

type Merge struct {
	MergePreference hash
	DataSources     []string
	MergeMode       string
	DataMutex       *sync.RWMutex
	Data            map[string][]hash
}

// Merge - Entry point from executor to perform merging of data from multiple sources.
// Internally it invokes source.Request, source.Extract and source.Normalize
func (mg *Merge) Merge(workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (code string, err error) {
	mg.Data = make(map[string][]hash, 0)
	mg.DataMutex = &sync.RWMutex{}
	sourceObjs := make([]types.Sources, 0)
	for _, source := range mg.DataSources {
		var sourceObj types.Sources
		if source == "M101" {
			sourceObj = &m101.M101{Name: source}
		} else if source == "WRAPPER" {
			sourceObj = &supervised.Supervised{Name: source}
		} else if source == "UNSUPERVISED" {
			sourceObj = &unsupervised.Unsupervised{Name: source}
		} else if source == "DIFFBOT" {
			sourceObj = &diffbot.Diffbot{Name: source}
		} else if source == "AMAZON" {
			sourceObj = &amazon.Amazon{Name: source}
		} else {
			log.Printf("UNKNOWN_DATA_SOURCE: Source %s, URL %s", source, workflow.URL)
		}
		sourceObjs = append(sourceObjs, sourceObj)
	}
	// Normalize each data source in for loop
	// Implement merge logic
	if mg.MergeMode == "MERGE_ALL" {
		code, err = mg.InitiateConcurrently(sourceObjs, workflow, pipeline, appC)
	} else {
		code, err = mg.InitiateSeq(sourceObjs, workflow, pipeline, appC)
	}
	return code, err
}

// InitiateConcurrently - Initiate all extract functions concurrently
// Store all data into an array and call merge data
func (mg *Merge) InitiateConcurrently(dataSources []types.Sources, workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (code string, err error) {
	var wg sync.WaitGroup
	for _, dataSource := range dataSources {
		wg.Add(1)
		go func(ds types.Sources, wc types.CrawlWorkflow) {
			defer wg.Done()
			canExtract, code, err := ds.Request(workflow.URL, workflow, pipeline, appC)
			log.Printf("CONCURRENT_REQUEST_RESULT: source %s, canExtract %v, code %s, error %v", ds.GetName(), canExtract, code, err)
			if err == nil && canExtract {
				code, err = ds.Extract(workflow.URL, workflow, pipeline, appC)
				log.Printf("CONCURRENT_EXTRACT_RESULT: source %s, code %s, error %v", ds.GetName(), code, err)
				if err == nil {
					ds.Normalize(workflow, appC)
					mg.DataMutex.Lock()
					mg.Data[ds.GetName()] = wc.Data.Products
					mg.DataMutex.Unlock()
				}
			}
		}(dataSource, *workflow)
	}

	wg.Wait()
	if mg.MergePreference == nil {
		for source := range mg.Data {
			mg.MergePreference = generateDefaultMergePreference(mg.Data[source], mg.DataSources)
			break
		}
	}
	// Individual product/variation merging logic
	// Merge individual products into an array and overwrite the workflow.Data.Products
	mg.TransformProductsAndMerge(workflow)
	return
}

// InitiateSeq - Initiate all extract functions sequentially (to save M101 API call costs- avoid if wrapper extracts data)
// Store all data into an array and call merge data
func (mg *Merge) InitiateSeq(dataSources []types.Sources, workflow *types.CrawlWorkflow, pipeline types.Pipeline, appC *types.Config) (code string, err error) {
	var canExtract bool
	for _, ds := range dataSources {

		// Track request metrics in Datadog for each data source
		ddMetrics := stats.DatadogMetrics{
			Source: ds.GetName(),
			Start:  time.Now(),
		}
		defer func(d types.Sources, metrics stats.DatadogMetrics) {
			metrics.Code = d.GetErrorCode()
			stats.WriteMetricsToDatadog(metrics, workflow, appC)
		}(ds, ddMetrics)

		canExtract, code, err = ds.Request(workflow.URL, workflow, pipeline, appC)
		ddMetrics.Code = code
		log.Printf("SEQUENTIAL_REQUEST_RESULT: source %s, canExtract %v, code %s, error %v", ds.GetName(), canExtract, code, err)
		// check if it is permanent failure
		if err != nil && (code == "NOT_PRODUCT_PAGE" || code == "DOES_NOT_EXIST") {
			workflow.Data.Code = code
			workflow.Data.Status = 0
			workflow.Data.Message = err.Error()
			break
		}

		if err == nil && canExtract {
			code, err = ds.Extract(workflow.URL, workflow, pipeline, appC)
			ddMetrics.Code = code
			log.Printf("SEQUENTIAL_EXTRACT_RESULT: source %s, code %s, error %v", ds.GetName(), code, err)
			// If the url is not a product page as detected by WRAPPER/SITEDETAILS we do not want
			// to fall back to other data sources
			if err != nil && (code == "NOT_PRODUCT_PAGE" || code == "DOES_NOT_EXIST") {
				workflow.Data.Code = code
				workflow.Data.Status = 0
				workflow.Data.Message = err.Error()
				break
			}
			if err == nil && len(workflow.Data.Products) > 0 {
				ds.Normalize(workflow, appC)
				mg.Data[ds.GetName()] = workflow.Data.Products
				// Send name of data source used
				workflow.Data.ExtractionDataSource = ds.GetName()
				// exiting loop to avoid making other API network calls
				break
			}
		}
	}

	// Only when the data was extracted from multiple sources, perform MERGING else return the result directly
	if len(mg.Data) > 1 {
		// Send in the first source's products array to generate mergePreference,
		// assuming the other sources products array is matching in order and normalized successfully
		for source := range mg.Data {
			mg.MergePreference = generateDefaultMergePreference(mg.Data[source], mg.DataSources)
			break
		}
		// Merge individual products into an array and overwrite the workflow.Data.Products
		mg.TransformProductsAndMerge(workflow)
	}
	return
}

// TransformProductsAndMerge - creates a local variable dfs to match the first variable parameter of function mergeData
// Assuming that all sources other than WRAPPER returns a single product array
func (mg *Merge) TransformProductsAndMerge(workflow *types.CrawlWorkflow) {
	// Fetching source with max products crawled to iterate over
	log.Printf("Transforming Products and Merging...\n")
	source := getMaxLengthKey(mg.Data)
	products := mg.Data[source]
	for i := 0; i < len(products); i++ {
		// To store a single product at a time but from multiple sources, e.g. WRAPPER,M101,etc.
		// dfs = {"WRAPPER":PRODUCT_1,"M101":PRODUCT_1,"UNSUPERVISED":PRODUCT_1}
		dfs := make(map[string]hash, 0)
		// Looping over all sources to fetch the product at index i
		for src := range mg.Data {
			if src != "WRAPPER" {
				// Since DIFFBOT, UCE and M101 returns a product response of length 1
				dfs[src] = mg.Data[src][0]
			} else {
				dfs[src] = mg.Data[src][i]
			}
		}
		merged, fieldSources := mergeData(dfs, mg.MergePreference)
		log.Printf("Product %d has the following fieldSources %v\n", i, fieldSources)
		workflow.Data.Products[i] = merged
	}
}

// getMaxLengthKey - Takes in all source's normalized products and returns the source name with maximum number of products in the array.
func getMaxLengthKey(data map[string][]hash) (key string) {
	maxLen := 0
	for src, products := range data {
		if len(products) >= maxLen {
			key = src
		}
	}
	return key
}

// generateDefaultMergePreference - for all Sem3-supported product keys return a mergePreference hash
// e.g {"name":["M101","WRAPPER"],"description":["M101","WRAPPER"],....}
func generateDefaultMergePreference(products []hash, dataSources []string) (mergePreference hash) {
	mergePreference = make(hash, 0)
	productKeys := []string{"_id", "sku", "time", "description", "listprice", "listprice_currency", "offers", "offers1", "offers2", "model", "images", "internal_fields", "is_active", "url", "crumb", "features", "name", "name_firstkeyword", "processing_fields", "_reserved_init_url", "crawl_id", "geo_id", "department", "siterating", "ean", "width_unit", "brand", "tracks", "department", "variation_tag", "isbn13", "isbn10", "isbn", "weight", "weight_unit", "recentoffers_count", "publisher", "published_at", "studio", "filmrating", "salesrank", "length_unit", "variation_ischild", "sizelookup", "reviews_number", "pages", "reviews_individual_number", "variation_ids", "variation_id", "upc", "images1", "images2", "height_unit", "author", "size", "color", "asin", "colorlookup", "upc14", "mpn", "height", "packagequantity", "artist", "length", "format", "siterating_scale", "images_count", "manufacturer"}
	for _, keys := range productKeys {
		mergePreference[keys] = dataSources
	}
	// for _, product := range products {
	// 	for key, _ := range product {
	// 		mergePreference[key] = dataSources
	// 	}
	// }
	return
}

// mergeData - dataFromSources {"WRAPPER":{name, desc}, "M101": {name, desc}} [data of 1 child variation from all sources]
func mergeData(dataFromSources map[string]hash, mergePreference hash) (merged hash, fieldSources hash) {
	// In case of Sequential, No need to merge since the data is fetched from a single source by avoiding other API calls
	if mergePreference == nil {
		return nil, nil
	}
	merged = make(hash, 0)
	fieldSources = make(hash, 0)
	for key, sources := range mergePreference {
		switch v := sources.(type) {
		case []string:
			for _, source := range sources.([]string) {
				if !isEmptyValue(dataFromSources[source][key]) {
					merged[key] = dataFromSources[source][key]
					fieldSources[key] = source
					break
				}
			}
		case string:
			source, _ := sources.(string)
			if !isEmptyValue(dataFromSources[source][key]) {
				merged[key] = dataFromSources[source][key]
				fieldSources[key] = source
			}
		default:
			log.Printf("MERGE_UNKNOWN_SOURCE_TYPE_SKIP: Source %v, Type %v", sources, v)
		}
	}
	return
}

// Returns true if the passed value is found to be empty and false otherwise
func isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	// empty generic array
	if v := reflect.ValueOf(value); (v.Kind() == reflect.Array || v.Kind() == reflect.Slice) && v.Len() == 0 {
		return true
	}
	switch v := value.(type) {
	case []string:
		return v == nil || len(v) == 0
	case []interface{}:
		return v == nil || len(v) == 0
	case map[string]interface{}:
		return v == nil || len(v) == 0
	case string:
		return v == ""
	default:
		return false
	}
}
