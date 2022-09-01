package stats

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/Semantics3/go-crawler/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

func initializeBatchExtractionMetrics(sm *types.StatsManager, em types.ExtractionMetrics) {

	if _, ok := sm.BatchExtractionMetrics[em.Customer]; !ok {
		sm.BatchExtractionMetrics[em.Customer] = make(map[string]map[string]map[string]interface{})
	}

	if _, ok := sm.BatchExtractionMetrics[em.Customer][em.Site]; !ok {
		sm.BatchExtractionMetrics[em.Customer][em.Site] = make(map[string]map[string]interface{})
	}

	if _, ok := sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]; !ok {
		sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType] = make(map[string]interface{})
	}

	intKeys := []string{"url_count", "iterations", "value"}
	floatKeys := []string{"s3", "products", "links", "preprocess", "total"}

	for _, k := range intKeys {
		if _, ok := sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType][k]; !ok {
			sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType][k] = 0
		}
	}

	for _, k := range floatKeys {
		if _, ok := sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType][k]; !ok {
			sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType][k] = 0.0
		}
	}
}

func aggregateExtractionMetrics(sm *types.StatsManager, em types.ExtractionMetrics) {
	s3 := sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["s3"].(float64) + em.S3
	products := sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["products"].(float64) + em.Products
	links := sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["links"].(float64) + em.Links
	preprocess := sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["preprocess"].(float64) + em.Preprocess
	total := sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["total"].(float64) + em.Total
	iterations := sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["iterations"].(int) + em.Iterations
	urlCount := sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["url_count"].(int) + em.UrlCount
	value := sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["value"].(int) + em.Value

	// Update batch stats
	sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["s3"] = s3
	sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["products"] = products
	sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["links"] = links
	sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["preprocess"] = preprocess
	sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["total"] = total
	sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["url_count"] = urlCount
	sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["iterations"] = iterations
	sm.BatchExtractionMetrics[em.Customer][em.Site][em.JobType]["value"] = value
}

func flushExtractionMetricsToDatabase(sm *types.StatsManager, appC *types.Config) {
	for customer, _ := range sm.BatchExtractionMetrics {
		for site, _ := range sm.BatchExtractionMetrics[customer] {
			for jobType, fields := range sm.BatchExtractionMetrics[customer][site] {
				// Construct tags
				tags := map[string]string{
					"customer": customer,
					"site":     site,
					"job_type": jobType,
				}
				tm := time.Now()
				writeDataToInfluxDB(appC.ConfigData.Influx.ExtractionMetrics, tags, fields, tm, appC)
			}
		}
	}

	return
}

func writeExtractionMetricsToDatadog(statsdClient *statsd.Client, metricName string, em *types.ExtractionMetrics) {

	extractionMetricsJson := make(map[string]interface{})
	pmBytes, _ := json.Marshal(em)
	json.Unmarshal(pmBytes, &extractionMetricsJson)

	// Construct tags
	tags := []string{
		fmt.Sprintf("site:%s", extractionMetricsJson["site"]),
		fmt.Sprintf("job_type:%s", extractionMetricsJson["job_type"]),
	}

	// Collect all the metrics for which we need max, min, avg values
	metrics := []string{"total", "s3", "products", "links", "preprocess"}
	for _, metric := range metrics {
		var fieldLevelMetricName string
		if metric == "s3" {
			fieldLevelMetricName = fmt.Sprintf("%s.%s", metricName, "cache")
		} else {
			fieldLevelMetricName = fmt.Sprintf("%s.%s", metricName, metric)
		}
		val := extractionMetricsJson[metric].(float64)
		statsdClient.Distribution(fieldLevelMetricName, val, tags, 1)
	}

	// Collect metrics for which we would need number of occurances
	var fieldLevelMetricName string
	counts := []string{"iterations", "url_count"}
	for _, count := range counts {
		switch count {
		case "url_count":
			fieldLevelMetricName = fmt.Sprintf("%s.%s", metricName, "ajax.count")
		case "iterations":
			fieldLevelMetricName = fmt.Sprintf("%s.%s", metricName, "iterations.count")
		}
		if val, ok := cutils.GetInt64Key(extractionMetricsJson, count); ok {
			statsdClient.Count(fieldLevelMetricName, val, tags, 1)
		}
	}

	// Increment request count
	fieldLevelMetricName = fmt.Sprintf("%s.%s", metricName, "requests.count")
	statsdClient.Incr(fieldLevelMetricName, tags, 1)
}
