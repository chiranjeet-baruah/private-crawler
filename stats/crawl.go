package stats

import (
	"fmt"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/Semantics3/go-crawler/types"
)

func initializeBatchCrawlMetrics(sm *types.StatsManager, cm types.CrawlMetrics) {

	if _, ok := sm.BatchCrawlMetrics[cm.Customer]; !ok {
		sm.BatchCrawlMetrics[cm.Customer] = make(map[string]map[string]map[string]map[string]map[string]map[string]map[string]map[string]interface{})
	}

	if _, ok := sm.BatchCrawlMetrics[cm.Customer][cm.Site]; !ok {
		sm.BatchCrawlMetrics[cm.Customer][cm.Site] = make(map[string]map[string]map[string]map[string]map[string]map[string]map[string]interface{})
	}

	if _, ok := sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType]; !ok {
		sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType] = make(map[string]map[string]map[string]map[string]map[string]map[string]interface{})
	}

	if _, ok := sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency]; !ok {
		sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency] = make(map[string]map[string]map[string]map[string]map[string]interface{})
	}

	if _, ok := sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool]; !ok {
		sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool] = make(map[string]map[string]map[string]map[string]interface{})
	}

	if _, ok := sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool]; !ok {
		sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool] = make(map[string]map[string]map[string]interface{})
	}

	if _, ok := sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax]; !ok {
		sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax] = make(map[string]map[string]interface{})
	}

	if _, ok := sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]; !ok {
		sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status] = make(map[string]interface{})
	}

	if _, ok := sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]["latency"]; !ok {
		sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]["latency"] = 0.0
	}

	if _, ok := sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]["value"]; !ok {
		sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]["value"] = 0
	}

	if _, ok := sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]["content_length"]; !ok {
		sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]["content_length"] = 0
	}
}

func aggregateCrawlMetrics(sm *types.StatsManager, cm types.CrawlMetrics) {
	batchLatency := sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]["latency"].(float64) + cm.Latency
	batchValue := sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]["value"].(int) + cm.Value
	batchContentLength := sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]["content_length"].(int) + cm.ContentLength

	// Update batch stats
	sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]["latency"] = batchLatency
	sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]["value"] = batchValue
	sm.BatchCrawlMetrics[cm.Customer][cm.Site][cm.JobType][cm.RecrawlFrequency][cm.NodePool][cm.RenderPool][cm.IsAjax][cm.Status]["content_length"] = batchContentLength
}

func flushCrawlMetricsToDatabase(sm *types.StatsManager, appC *types.Config) {
	for customer, _ := range sm.BatchCrawlMetrics {
		for site, _ := range sm.BatchCrawlMetrics[customer] {
			for jobType, _ := range sm.BatchCrawlMetrics[customer][site] {
				for recrawlFrequency, _ := range sm.BatchCrawlMetrics[customer][site][jobType] {
					for nodePool, _ := range sm.BatchCrawlMetrics[customer][site][jobType][recrawlFrequency] {
						for renderPool, _ := range sm.BatchCrawlMetrics[customer][site][jobType][recrawlFrequency][nodePool] {
							for isAjax, _ := range sm.BatchCrawlMetrics[customer][site][jobType][recrawlFrequency][nodePool][renderPool] {
								for status, fields := range sm.BatchCrawlMetrics[customer][site][jobType][recrawlFrequency][nodePool][renderPool][isAjax] {

									// Construct tags
									tags := map[string]string{
										"customer":          customer,
										"site":              site,
										"job_type":          jobType,
										"recrawl_frequency": recrawlFrequency,
										"node_pool":         nodePool,
										"render_pool":       renderPool,
										"is_ajax":           isAjax,
										"status":            status,
									}
									tm := time.Now()
									writeDataToInfluxDB(appC.ConfigData.Influx.CrawlMetrics, tags, fields, tm, appC)
								}
							}
						}
					}
				}
			}
		}
	}

	return
}

func writeCrawlMetricsToDatadog(statsdClient *statsd.Client, metricName string, cm *types.CrawlMetrics) {
	tags := []string{
		fmt.Sprintf("site:%s", cm.Site),
		fmt.Sprintf("job_type:%s", cm.JobType),
		fmt.Sprintf("node_pool:%s", cm.NodePool),
		fmt.Sprintf("render_pool:%s", cm.RenderPool),
		fmt.Sprintf("status:%s", cm.Status),
	}

	var fieldLevelMetricName string

	fieldLevelMetricName = fmt.Sprintf("%s.%s", metricName, "latency")
	val := float64(cm.Latency)
	statsdClient.Distribution(fieldLevelMetricName, val, tags, 1)

	fieldLevelMetricName = fmt.Sprintf("%s.%s", metricName, "content_length")
	val = float64(cm.ContentLength)
	statsdClient.Distribution(fieldLevelMetricName, val, tags, 1)

	// Increment request count
	fieldLevelMetricName = fmt.Sprintf("%s.%s", metricName, "requests.count")
	statsdClient.Incr(fieldLevelMetricName, tags, 1)
}
