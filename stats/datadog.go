package stats

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
)

type DatadogMetrics struct {
	Source string
	Code   string
	Start  time.Time
}

func WriteMetricsToDatadog(metrics DatadogMetrics, workflow *types.CrawlWorkflow, appC *types.Config) {
	if metrics.Source == "WRAPPER" {
		workflow.ProductMetrics.ErrorCode = metrics.Code
		sendWrapperMetricsToDatadog(workflow, appC, metrics.Start)
		utils.UpdateExtractionMetrics(workflow, appC)
	} else {
		// Increment request metrics only if we actually make a request
		// It is possible for the data sources to return without making actual request
		// because of not being able to acquire lock in internal rate limiting
		if !strings.Contains(metrics.Code, "RATELIMIT") {
			count := 1
			sendDataSourceMetricsToDatadog(appC.StatsdClient, metrics.Source, workflow.DomainInfo.DomainName, workflow.ProductMetrics.JobType, metrics.Code, count, metrics.Start)
		}
	}
}

// Write wrapper data source metrics to datadog
func sendWrapperMetricsToDatadog(workflow *types.CrawlWorkflow, appC *types.Config, start time.Time) {

	pm := workflow.ProductMetrics
	statsdClient := appC.StatsdClient

	// Update metrics with necessary fields
	pm.Total = utils.ComputeDuration(start)
	if workflow.Status == 0 && workflow.FailureType != nil {
		pm.ErrorCode = *workflow.FailureType
	}

	ddMetricName := "crawler.data_source"

	// Construct tags
	tags := []string{
		"source:wrapper",
		fmt.Sprintf("site:%s", pm.Site),
		fmt.Sprintf("job_type:%s", pm.JobType),
	}

	// Collect all the metrics for which we need max, min, avg values
	fieldLevelMetricName := fmt.Sprintf("%s.%s", ddMetricName, "duration")
	statsdClient.Distribution(fieldLevelMetricName, pm.Total, tags, 1)

	fieldLevelMetricName = fmt.Sprintf("%s.%s", ddMetricName, "latency")
	statsdClient.Distribution(fieldLevelMetricName, pm.Latency, tags, 1)

	fieldLevelMetricName = fmt.Sprintf("%s.%s", ddMetricName, "extraction")
	statsdClient.Distribution(fieldLevelMetricName, pm.Extraction, tags, 1)

	// Increment request count
	tags = append(tags, fmt.Sprintf("error:%s", pm.ErrorCode))
	fieldLevelMetricName = fmt.Sprintf("%s.%s", ddMetricName, "requests.count")
	log.Printf("DATADOG metric: %s, tags: %v\n", fieldLevelMetricName, tags)
	statsdClient.Incr(fieldLevelMetricName, tags, 1)
}

// Write metrics for different (non wrapper) data sources to datadog
func sendDataSourceMetricsToDatadog(statsdClient *statsd.Client, source, site, job_type, errorCode string, count int, start time.Time) {

	metricName := "crawler.data_source"
	tags := []string{
		fmt.Sprintf("source:%s", source),
		fmt.Sprintf("site:%s", site),
		fmt.Sprintf("job_type:%s", job_type),
	}

	// 1. Track duration
	duration := utils.ComputeDuration(start)
	fieldLevelMetricName := fmt.Sprintf("%s.%s", metricName, "duration")
	log.Printf("DATADOG metric: %s, tags: %v, duration: %f\n", fieldLevelMetricName, tags, duration)
	statsdClient.Distribution(fieldLevelMetricName, duration, tags, float64(count))

	// 2. Request count

	// Add error code as tag
	// Remove any data source specific prefixes from code as we're adding it as a tag
	identifiers := []string{"UNSUPERVISED_", "DIFFBOT_", "AMAZON_", "M101_"}
	for _, identifier := range identifiers {
		errorCode = strings.Replace(errorCode, identifier, "", -1)
	}
	tags = append(tags, fmt.Sprintf("error:%s", errorCode))

	fieldLevelMetricName = fmt.Sprintf("%s.%s", metricName, "requests.count")
	log.Printf("DATADOG metric: %s, tags: %v\n", fieldLevelMetricName, tags)
	statsdClient.Incr(fieldLevelMetricName, tags, float64(count))
}
