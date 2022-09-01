package helper

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	"github.com/Semantics3/sem3-go-data-consumer/consume"
)

func ListenOnQueue(consumer *consume.Consumer, consumeFn consume.WorkFnType, queueConfig *types.QueueConfig, brokerURL string, etlController string, env string) {
	options := consume.ConsumerOptions{
		Durable:       true,
		PrefetchCount: queueConfig.PrefetchCount,
		ServiceName:   queueConfig.ServiceName,
		Mode:          env,
		NoStats:       true,
	}
	consumer.Init(queueConfig.QueueName, options, consumeFn, brokerURL, etlController)

	err := consumer.Consume()
	if err != nil {
		log.Printf("SERVICE_RMQ_CONSUMER_ERR: Error consuming messages for %s : %v\n", queueConfig.QueueName, err)
		os.Exit(1)
	}
}

func ConsumerCrawlFn(appC *types.Config, queueName string) consume.WorkFnType {
	return func(message []byte) (workResult consume.WorkResult) {
		workResult = consume.WorkResult{
			Success:      false,
			ErrorType:    consume.NonRecoverable,
			ErrorMessage: "Not executed",
		}

		// TODO: 1. Validate input

		// 2. Parse input
		batch := &ctypes.Batch{}
		err := json.Unmarshal(message, &batch)
		if err != nil {
			workResult.ErrorMessage = fmt.Sprintf("CRAWL_CONSUMER_MSG_UNMARSHAL_ERR: Error decoding message json: %v\n", err)
			return workResult
		}
		workResult.MsgId = batch.BatchID
		workResult.MsgVal = message

		// Extract domain name for current task
		var tasksResults ctypes.TasksResults
		for url := range batch.Tasks {
			site, err := utils.GetDomainName(url, appC.ConfigData.WrapperServiceURI)
			if err != nil {
				workResult.ErrorMessage = fmt.Sprintf("CRAWL_CONSUMER_ERR: Extracting domain name from %s faile with error: %v", url, err)
				return workResult
			}

			if pool, ok := appC.ConsumerSitePoolMap[site]; ok {
				// size: Will always return concurrency set during pool creation (constant)
				// length: Will return no.of items currently present in queue (variable)
				size := int64(pool.GetSize())
				length := pool.QueueLength()
				if length > size {
					log.Printf("CONSUMER_CONCURRENCY: Site level concurrency threshold met. Pushing item back to queue. URL: %s, QUEUE_SIZE: %d, CURRENT_QUEUE_LENGTH: %d\n", url, size, length)
					time.Sleep(2 * time.Second)
					publishToQueue(url, queueName, batch, appC)
				} else {
					// Synchronous operation (blocking)
					// Assumption here is batch will have only 1 item
					log.Printf("CONSUMER_CONCURRENCY: Adding item to queue. URL: %s, QUEUE_SIZE: %d, CURRENT_QUEUE_LENGTH: %d\n", url, size, length)
					taskData := map[string]interface{}{"batch": batch, "queue": queueName}
					results := pool.Process(taskData)
					tasksResults = results.(ctypes.TasksResults)
				}
			} else {
				tasksResults, _, err = CrawlJobBatchExecute(batch, appC, queueName)
			}
		}

		workResult.ErrorCode = ""
		workResult.ErrorMessage = ""
		workResult.Success = true
		workResult.Response = tasksResults
		return workResult
	}
}

func publishToQueue(url string, queue string, batch *ctypes.Batch, appC *types.Config) {
	var err error
	switch queue {
	case "crawl_worker_publisher_queue":
		err = appC.Publishers.CrawlWorkerPublisher.Publish(url, batch)
	}
	if err != nil {
		log.Printf("ST_PUBLISH_ERR: Error while publishing (%s) to (%s)", url, queue)
	}
}
