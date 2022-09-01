package stats

import (
	"fmt"
	"log"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	influx "github.com/influxdata/influxdb/client/v2"
)

func InitializeStatsManagerClient(sm *types.StatsManager) {
	sm.BatchCrawlMetrics = make(types.BatchCrawlMetrics)
	sm.BatchProductMetrics = make(types.BatchProductMetrics)
	sm.BatchExtractionMetrics = make(types.BatchExtractionMetrics)
	sm.CrawlMetricsChannel = make(chan types.CrawlMetrics, 250)
	sm.ProductMetricsChannel = make(chan types.ProductMetrics, 250)
	sm.ExtractionMetricsChannel = make(chan types.ExtractionMetrics, 250)
	sm.JobServerBatchStatsChannel = make(chan types.JobServerBatchStats, 250)
	sm.LastFlushTime = time.Now()
	sm.Init = true
}

func CollectStats(env string, sm *types.StatsManager, appC *types.Config) {
	influxTm := 600
	if env == "dev" || env == "staging" {
		influxTm = 60
	}
	tickerInflux := time.NewTicker(time.Second * time.Duration(influxTm))

	for {
		select {

		// Channel to receive crawl metrics
		case cm := <-sm.CrawlMetricsChannel:
			// utils.PrettyJSON("STATS_1001: CRAWL_METRICS", cm, true)
			initializeBatchCrawlMetrics(sm, cm)
			aggregateCrawlMetrics(sm, cm)
			writeCrawlMetricsToDatadog(appC.StatsdClient, appC.ConfigData.Influx.CrawlMetrics, &cm)

		// Channel to receive product metrics
		case pm := <-sm.ProductMetricsChannel:
			// utils.PrettyJSON("STATS_1002: PRODUCT_METRICS", pm, true)
			initializeBatchProductMetrics(sm, pm)
			aggregateProductMetrics(sm, pm)

		// Channel to receive extraction metrics
		case em := <-sm.ExtractionMetricsChannel:
			// utils.PrettyJSON("STATS_1003: EXTRACTION_METRICS", em, true)
			initializeBatchExtractionMetrics(sm, em)
			aggregateExtractionMetrics(sm, em)
			writeExtractionMetricsToDatadog(appC.StatsdClient, appC.ConfigData.Influx.ExtractionMetrics, &em)

		// Channel to receive product metrics
		case bs := <-sm.JobServerBatchStatsChannel:
			utils.PrettyJSON("STATS_1003: BATCH_STATS", bs, true)
			writeJobServerBatchStatsToInflux(bs, appC)

		// Channel to flush cached stats to influxdb: every 5 mins
		case _ = <-tickerInflux.C:
			log.Println("STATS_1004: Flushing stats to influxdb at 10-min interval")
			flushCrawlMetricsToDatabase(sm, appC)
			flushProductMetricsToDatabase(sm, appC)
			flushExtractionMetricsToDatabase(sm, appC)
			clearMetrics(sm)
		}
	}
}

func writeJobServerBatchStatsToInflux(jobServerBatchStats types.JobServerBatchStats, appC *types.Config) {
	tags := map[string]string{
		"site":              jobServerBatchStats.Site,
		"customer":          jobServerBatchStats.Customer,
		"job_type":          jobServerBatchStats.JobType,
		"recrawl_frequency": jobServerBatchStats.RecrawlFrequency,
	}

	fields := map[string]interface{}{
		"batch_size": jobServerBatchStats.BatchSize,
		"duration":   jobServerBatchStats.Duration,
		"value":      jobServerBatchStats.Value,
	}

	writeDataToInfluxDB("job_server_batch_stats_v2", tags, fields, time.Now(), appC)
}

func writeDataToInfluxDB(measurement string, tags map[string]string, fields map[string]interface{}, tm time.Time, appC *types.Config) {
	// Make client
	config := influx.UDPConfig{Addr: appC.ConfigData.Influx.Server}
	c, err := influx.NewUDPClient(config)
	if err != nil {
		log.Println("ERROR: Error creating UDP client for influxdb: ", err.Error())
		return
	}
	defer c.Close()

	log.Printf("INFLUX_1001: Measurement: %s, tags: %v, fields: %v, tm: %v, InfluxUrl: %s\n", measurement, tags, fields, tm, appC.ConfigData.Influx.Server)

	// Create a new point batch
	bp, _ := influx.NewBatchPoints(influx.BatchPointsConfig{
		Precision: "ns",
	})

	// Create a point and add to batch
	pt, err := influx.NewPoint(measurement, tags, fields, tm)
	if err != nil {
		panic(err.Error())
	}
	bp.AddPoint(pt)

	// Write the batch
	c.Write(bp)
}

func clearMetrics(sm *types.StatsManager) {
	log.Println("STATS_1401: Clearing crawl and product metrics")
	for key, _ := range sm.BatchCrawlMetrics {
		delete(sm.BatchCrawlMetrics, key)
	}
	for key, _ := range sm.BatchProductMetrics {
		delete(sm.BatchProductMetrics, key)
	}
	for key, _ := range sm.BatchExtractionMetrics {
		delete(sm.BatchExtractionMetrics, key)
	}
}

func writeCustomMetricsToDataDog(statsdClient *statsd.Client, metricName string, tags map[string]string, fields map[string]interface{}) {
	metrics := make([]string, 0)

	for key, value := range tags {
		metrics = append(metrics, fmt.Sprintf("%s:%s", key, value))
	}

	for key, value := range fields {
		metrics = append(metrics, fmt.Sprintf("%s:%s", key, fmt.Sprint(value)))
	}

	// Example metric
	// statsd.Incr("example_metric.increment", []string{"environment:sem3stage"}, 1)
	log.Printf("Flushing stats to Datadog for %s: %v\n", metricName, metrics)
	statsdClient.Incr(metricName, metrics, 1)
}
