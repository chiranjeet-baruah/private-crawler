package stats

import (
	"time"

	"github.com/Semantics3/go-crawler/types"
)

func initializeBatchProductMetrics(sm *types.StatsManager, pm types.ProductMetrics) {

	if _, ok := sm.BatchProductMetrics[pm.Customer]; !ok {
		sm.BatchProductMetrics[pm.Customer] = make(map[string]map[string]map[string]map[string]map[string]interface{})
	}

	if _, ok := sm.BatchProductMetrics[pm.Customer][pm.Site]; !ok {
		sm.BatchProductMetrics[pm.Customer][pm.Site] = make(map[string]map[string]map[string]map[string]interface{})
	}

	if _, ok := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType]; !ok {
		sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType] = make(map[string]map[string]map[string]interface{})
	}

	if _, ok := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency]; !ok {
		sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency] = make(map[string]map[string]interface{})
	}

	if _, ok := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]; !ok {
		sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success] = make(map[string]interface{})
	}

	if _, ok := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["total"]; !ok {
		sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["total"] = 0.0
	}

	if _, ok := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["latency"]; !ok {
		sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["latency"] = 0.0
	}
	if _, ok := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["domain_info"]; !ok {
		sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["domain_info"] = 0.0
	}

	if _, ok := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["extraction"]; !ok {
		sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["extraction"] = 0.0
	}

	if _, ok := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["url_count"]; !ok {
		sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["url_count"] = 0
	}

	if _, ok := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["retry_count"]; !ok {
		sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["retry_count"] = 0
	}

	if _, ok := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["value"]; !ok {
		sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["value"] = 0
	}
}

func aggregateProductMetrics(sm *types.StatsManager, pm types.ProductMetrics) {
	batchTotal := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["total"].(float64) + pm.Total
	batchExtraction := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["extraction"].(float64) + pm.Extraction
	batchLatency := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["latency"].(float64) + pm.Latency
	batchDomainInfo := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["domain_info"].(float64) + pm.DomainInfo
	batchUrlCount := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["url_count"].(int) + pm.UrlCount
	batchValue := sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["value"].(int) + pm.Value

	// Update batch stats
	sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["total"] = batchTotal
	sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["latency"] = batchLatency
	sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["url_count"] = batchUrlCount
	sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["extraction"] = batchExtraction
	sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["domain_info"] = batchDomainInfo
	sm.BatchProductMetrics[pm.Customer][pm.Site][pm.JobType][pm.RecrawlFrequency][pm.Success]["value"] = batchValue
}

func flushProductMetricsToDatabase(sm *types.StatsManager, appC *types.Config) {
	for customer, _ := range sm.BatchProductMetrics {
		for site, _ := range sm.BatchProductMetrics[customer] {
			for jobType, _ := range sm.BatchProductMetrics[customer][site] {
				for recrawlFrequency, _ := range sm.BatchProductMetrics[customer][site][jobType] {
					for success, fields := range sm.BatchProductMetrics[customer][site][jobType][recrawlFrequency] {

						// Construct tags
						tags := map[string]string{
							"customer":          customer,
							"site":              site,
							"job_type":          jobType,
							"recrawl_frequency": recrawlFrequency,
							"success":           success,
						}
						tm := time.Now()
						writeDataToInfluxDB(appC.ConfigData.Influx.ProductMetrics, tags, fields, tm, appC)
					}
				}
			}
		}
	}
	return
}
