package helper

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/Semantics3/go-crawler/pipeline"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	jobutils "github.com/Semantics3/sem3-go-crawl-utils/jobs"
	redisutils "github.com/Semantics3/sem3-go-crawl-utils/redis"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

type crawlResult struct {
	url      string
	workflow *types.CrawlWorkflow
}

// Execute a batch of tasks from a job in parallel
func CrawlJobBatchExecute(jobInput *ctypes.Batch, appC *types.Config, queueName string) (tasksResults ctypes.TasksResults, crawlResults map[string]*types.CrawlWorkflow, err error) {
	batchSize := len(jobInput.Tasks)
	jobType := jobutils.GetJobType(jobInput)
	start := time.Now()
	log.Printf("CRAWLJOBEXECUTE_NEWBATCH: (JobType %s, BatchId %s) \n", jobType, jobInput.BatchID)

	inputCh := make(chan string, batchSize)
	outputCh := make(chan *crawlResult, batchSize)

	// Execute appropriate job pipeline in parallel: 12 workers
	numWorkers := 12
	if batchSize < numWorkers {
		numWorkers = batchSize
	}
	for id := 1; id <= numWorkers; id++ {
		go CrawlJobWorker(id, jobInput, inputCh, outputCh, appC, queueName)
	}
	for u, _ := range jobInput.Tasks {
		inputCh <- u
	}
	close(inputCh)

	tasksResults, crawlResults = CollectResultsAndAnalyze(jobInput, outputCh, start, appC)
	log.Printf("BATCH_END: (Service %s)\n", queueName)

	// Read translation stats from redis and update job_params of job after job completion
	isTranslateCrawl := false
	for _, workflow := range crawlResults {
		isTranslateCrawl = workflow.IsTranslateCrawl
		if isTranslateCrawl {
			break
		}
	}
	if isTranslateCrawl {
		go func(ji *ctypes.Batch, ac *types.Config) {
			err := UpdateJobTranslationStats(ji, ac)
			log.Printf("%v\n", err)
		}(jobInput, appC)
	} else {
		log.Printf("NOT_STARTING_TRANSLATE_STATS_UPDATER\n")
	}

	return tasksResults, crawlResults, err
}

// Job worker
func CrawlJobWorker(id int, jobInput *ctypes.Batch, inputCh chan string, outputCh chan *crawlResult, appC *types.Config, queueName string) {
	for url := range inputCh {
		log.Printf("JOB_WORKER_BEGIN: (Worker %d) (Service %s) %s\n", id, queueName, url)
		var workflow *types.CrawlWorkflow
		jobType := jobutils.GetJobType(jobInput)
		// jobTypeRegex, _ := regexp.Compile(`^(?:crawl|wrapperqa)`)
		ondemandCrawlJobTypeRegex, _ := regexp.Compile(`^(?:ondemand_crawl|ondemand_slow_crawl)`)
		var pipelineObj types.Pipeline
		if jobType == "testwrapper" {
			pipelineObj = &pipeline.TestWrapperPipeline{}
		} else if jobType == "crawl" {
			pipelineObj = &pipeline.CrawlPipeline{}
		} else if jobType == "recrawl" {
			pipelineObj = &pipeline.RecrawlPipeline{}
		} else if jobType == "realtimeapi" || strings.Contains(jobType, "webhooks") {
			pipelineObj = &pipeline.RealtimeApiPipeline{}
		} else if jobType == "wrapperqa" {
			pipelineObj = &pipeline.WrapperQAPipeline{}
		} else if ondemandCrawlJobTypeRegex.MatchString(jobType) {
			pipelineObj = &pipeline.OnDemandCrawlPipeline{}
		} else if jobType == "discovery_crawl" {
			pipelineObj = &pipeline.DiscoveryPipeline{}
		}
		workflow = pipeline.PipelineExecutor(url, jobInput, pipelineObj, appC, queueName)
		if workflow.FailureType != nil && workflow.FailureMessage != nil {
			code, err := pipelineObj.TransformError(*workflow.FailureType, fmt.Errorf("%s", *workflow.FailureMessage))
			utils.FailWorkflow(url, pipelineObj, workflow, code, err.Error(), appC)
		}
		outputCh <- &crawlResult{
			url:      url,
			workflow: workflow,
		}
	}
	log.Printf("JOB_WORKER_END: (Worker %d) (Service %s) Quitting\n", id, queueName)
}

// CollectResultsAndAnalyze will collect crawl results and aggregates success/failure stats
func CollectResultsAndAnalyze(jobInput *ctypes.Batch,
	outputCh chan *crawlResult,
	start time.Time,
	appC *types.Config) (tasksResults ctypes.TasksResults, crawlResults map[string]*types.CrawlWorkflow) {

	batchSize := len(jobInput.Tasks)
	tasksResults = make(ctypes.TasksResults, 0)
	crawlResults = make(map[string]*types.CrawlWorkflow, 0)
	crawlResultsStats := map[string]int{"success": 0, "failures": 0}
	var sampleWorkflow *types.CrawlWorkflow

	// Construct response to jobserver
	for i := 0; i < batchSize; i++ {
		crawlResult := <-outputCh
		workflow := crawlResult.workflow
		msg := fmt.Sprintf("URL: %s, PRODUCT_METRICS", workflow.URL)
		utils.PrettyJSON(msg, workflow.ProductMetrics, true)
		taskResult := make(map[string]interface{}, 0)
		taskResult["status"] = workflow.Status
		if workflow.Status == 0 && workflow.FailureType != nil {
			taskResult["status_failed_reason_type"] = *workflow.FailureType
			taskResult["status_failed_reason_message"] = *workflow.FailureMessage
		}

		// Construct jobserver feedback
		if (workflow.Status == 1 || workflow.SendFailureAsFeedback) && len(workflow.Data.Links) > 0 {
			taskFeedback := make(map[string]interface{}, 0)
			for task, metadata := range workflow.Data.Links {
				taskFeedback[task] = types.JobServerFeedback{
					Metadata: metadata,
					Priority: metadata.Priority,
				}
			}
			taskResult["feedback"] = taskFeedback
			log.Printf("URL: %s JOBSERVER_FEEDBACK_LINKS: %d\n", workflow.URL, len(taskFeedback))
		}

		tasksResults[crawlResult.url] = taskResult
		crawlResults[crawlResult.url] = workflow

		// Compute batch stats
		if workflow.Status == 1 {
			crawlResultsStats["success"]++
		} else {
			crawlResultsStats["failures"]++
			ft := workflow.FailureType
			if ft != nil {
				if _, ok := crawlResultsStats[*ft]; !ok {
					crawlResultsStats[*ft] = 0
				}
				crawlResultsStats[*ft]++
			}
		}

		sampleWorkflow = workflow
	}

	duration := utils.ComputeDuration(start)
	// utils.PrettyJSON("CRAWLJOB_BATCH_STATS:", crawlResultsStats, true)
	log.Printf("CRAWLJOB_BATCH_COMPLETE: (BatchId %s, Duration %f seconds) \n", jobInput.BatchID, duration)
	utils.UpdateJobServerBatchStats(batchSize, sampleWorkflow, duration, appC)
	return tasksResults, crawlResults
}

// Construct job batch from a url and job_type
func JobBatchFromUrls(urls []string, jobType string, jobId string, jobParams map[string]interface{}) (batch *ctypes.Batch) {
	if jobId == "" {
		jobId = "clitest"
	}
	jobId = fmt.Sprintf("%s_%s", jobType, jobId)
	batchId := fmt.Sprintf("%s_batch1", jobId)

	// Create job_params object
	if jobParams == nil {
		jobParams = make(map[string]interface{}, 0)
	}

	batch = &ctypes.Batch{
		JobID:     jobId,
		BatchID:   batchId,
		JobParams: jobParams,
		JobDetails: ctypes.JobConfig{
			JobType: jobType,
			State: &ctypes.JobState{
				TimeCreated: (time.Now().Unix() * 1000),
			},
		},
	}
	batch.Tasks = make(map[string]ctypes.UrlMetadata, 0)
	for _, url := range urls {
		batch.Tasks[url] = ctypes.UrlMetadata{
			Priority: 101,
			LinkType: "content",
		}
	}
	return batch
}

// UpdateJobTranslationStats - Read job's translation stats from redis and push to job
func UpdateJobTranslationStats(jobInput *ctypes.Batch, appC *types.Config) (err error) {
	defer func() {
		rand.Seed(time.Now().UTC().UnixNano())
	}()

	// Seed with batch_id so that job workers that reach this function
	// at the same time will get different rand times to sleep before
	// checking job-config (to decide whether to update translation stats)
	rand.Seed(utils.GetMD5Sum(jobInput.BatchID))

	jobServerUrl := appC.ConfigData.JobServer

	slpTime := 5 + rand.Intn(10)
	// log.Printf("UPDATE_JOB_TRANSLATION_STATS_SLEEP: Sleeping %d seconds\n", slpTime)
	time.Sleep(time.Duration(slpTime) * time.Second)

	// Fetch job config
	jobID := jobInput.JobID
	var config *ctypes.JobConfig
	// Job-server retries
	retries := 3
	for i := 0; i < retries; i++ {
		config, err = jobutils.GetJobStatus(jobID, jobServerUrl)
		if err != nil {
			log.Printf("JOB_SERVER_RETRY: %d\n", i)
			time.Sleep(time.Duration(slpTime) * time.Second)
			continue
		}
		// Successfully fetched job config, so quit retry loop
		break
	}
	if err != nil {
		return err
	}

	// If queued > 0 , quit immediately since stats are not finalized now
	if config.State.Queued > 0 {
		return nil
	}

	// If google api translation stats already updated in job, quit function
	_, isStatsUpdated := config.JobParams["google_translate_api_calls"]
	if isStatsUpdated {
		return nil
	}

	if config.State.PendingBatches > 5 {
		return nil
	}

	// Read stats from redis and Perform the job params update
	hashName := fmt.Sprintf("job_translation_stats;%s", jobID)
	var ts map[string]string
	for i := 0; i < retries; i++ {
		ts, err = redisutils.HGetAll(appC.RedisCrawl, hashName)
		if err != nil {
			log.Printf("REDIS_GET_TRANSLATION_STATS_RETRY: %d\n", i)
			time.Sleep(time.Duration(slpTime) * time.Second)
			continue
		}
		// Successfully fetched job config, so quit retry loop
		break
	}
	if err != nil {
		return fmt.Errorf("REDIS_GET_TRANSLATION_STATS_ERR: %v", err)
	}
	translationStats := map[string]interface{}{
		"google_translate_api_calls": ts,
	}

	jobParamsUrl := fmt.Sprintf("http://%s/jobs/%s/params", jobServerUrl, jobID)
	log.Printf("PATCH_JOBPARAMS_START: (%s) Patching job params with translation stats\n", jobParamsUrl)
	body, err := jobutils.RequestUrl("PATCH", jobParamsUrl, translationStats, "")
	if err != nil {
		err = cutils.PrintErr("PATCH_JOBPARAMS_ERR", fmt.Sprintf("failed to %s %s (%v)", "PATCH", jobParamsUrl, translationStats), err)
		return err
	}
	jobParamsResp := make(map[string]interface{}, 0)
	err = json.Unmarshal(body, &jobParamsResp)
	if err != nil {
		err = cutils.PrintErr("PATCH_JOBPARAMS_DECODE_ERR", fmt.Sprintf("failed to decode json %s %s (%v) (resp %v)", "PATCH", jobParamsUrl, translationStats, string(body)), err)
		return err
	}

	code, _ := jobParamsResp["code"]
	if code != "" {
		return fmt.Errorf("%s", string(body))
	}
	return nil
}
