package dbs

import (
	"fmt"
	"log"

	"github.com/Jeffail/tunny"
	servicehelper "github.com/Semantics3/go-crawler/service/helper"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
	"github.com/Semantics3/sem3-go-data-consumer/consume"
	publish "github.com/Semantics3/sem3-go-data-consumer/publish"
)

// CreateConsumerConnections will create consumer connections needed for pipeline based on env
func CreateConsumerConnections(cliArgs *types.CliArgs, configData types.ConfigData, appC *types.Config) (err error) {
	env := cliArgs.Env

	// Check if queue configs are set
	if configData.ConsumeQueueConfig == nil {
		utils.PrettyJSON(fmt.Sprintf("%s_config", env), configData, true)
		err = cutils.PrintErr("DBS_RMQ_CONSUMER_ERR", fmt.Sprintf("queue config data missing for %s mode\n", env), "")
		return err
	}

	// Create consumer site pool map to handle concurrency at site level
	if appC.ConfigData.ConsumerSitePoolConfig != nil && appC.ConsumerSitePoolMap == nil {
		appC.ConsumerSitePoolMap = make(map[string]*tunny.Pool)
		for site, concurrency := range appC.ConfigData.ConsumerSitePoolConfig {
			appC.ConsumerSitePoolMap[site] = tunny.NewFunc(concurrency, func(input interface{}) interface{} {
				data := input.(map[string]interface{})
				taskBatch := data["batch"].(*ctypes.Batch)
				taskQueue := data["queue"].(string)
				tasksResults, _, err := servicehelper.CrawlJobBatchExecute(taskBatch, appC, taskQueue)

				// TODO: Should this error be communicated upstream ?
				log.Println(err)

				return tasksResults
			})
		}
	}

	// Connect to publish queues
	err = connectToPublishQueues(configData, appC)
	if err != nil {
		return err
	}

	// Connect to consume queues
	for _, queueConfig := range configData.ConsumeQueueConfig {
		consumer := &consume.Consumer{}
		consumerWorkFn := servicehelper.ConsumerCrawlFn(appC, queueConfig.ServiceName)
		go func(c *consume.Consumer, q types.QueueConfig) {
			servicehelper.ListenOnQueue(c, consumerWorkFn, &q, configData.RdMsgBroker, configData.RdEtlController, env)
		}(consumer, queueConfig)
	}
	return nil
}

func connectToPublishQueues(configData types.ConfigData, appC *types.Config) (err error) {
	for _, queueConfig := range configData.PublishQueueConfig {
		var publisher publish.Publisher
		_, err = publisher.Connect(configData.RdMsgBroker, queueConfig.QueueName, true)
		if err != nil {
			err = cutils.PrintErr("DBS_RMQERR", fmt.Sprintf("failed to connect to publish queue (%s, %s)", configData.RdMsgBroker, queueConfig.QueueName), err)
			return err
		}
		initPublishQueue(queueConfig.ServiceName, &publisher, appC)
	}
	return nil
}

func initPublishQueue(serviceName string, publisher *publish.Publisher, appC *types.Config) {

	if appC.Publishers == nil {
		appC.Publishers = &types.PublisherConfig{}
	}

	// If the current worker needs to publish items to multiple queues,
	// multiple publishers must be created
	// Just adding the placeholders here
	switch serviceName {
	case "crawl_worker_publisher_queue":
		appC.Publishers.CrawlWorkerPublisher = publisher
	}
}
