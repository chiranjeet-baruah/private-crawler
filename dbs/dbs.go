package dbs

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	mongo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Semantics3/go-crawler/stats"
	"github.com/Semantics3/go-crawler/types"

	// jobutils "github.com/Semantics3/sem3-go-crawl-utils/jobs"
	redisutils "github.com/Semantics3/sem3-go-crawl-utils/redis"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
	validatelib "github.com/Semantics3/sem3-go-crawl-utils/validate"
	s3Cache "github.com/Semantics3/sem3-go-crawl-utils/webcache/s3"
	"github.com/Semantics3/sem3-go-data-consumer/consume"
	"github.com/Semantics3/sem3-go-data-consumer/publish"
	s3rpc "github.com/Semantics3/sem3-go-data-consumer/rpc"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/go-pg/pg"
)

// ParseCliArgs will parse client command line arguments
func ParseCliArgs() (cliArgs *types.CliArgs) {
	env := flag.String("env", "staging", "Which mode go crawl worker has to run in")
	prof := flag.Bool("pprof", false, "Whether to run pprof on the server for debugging")
	job := flag.Bool("job", false, "Whether to expose crawling as a job")
	rest := flag.Bool("rest", false, "Whether to expose crawling as a REST service")
	test := flag.Bool("test", false, "Whether to run in test mode")
	testFile := flag.Bool("test-file", false, "Whether to run in test file mode")
	jobType := flag.String("job-type", "recrawl", "Which job type to run crawl as")
	url := flag.String("url", "", "Url to test")
	filename := flag.String("file", "", "file containing urls to test")
	consume := flag.Bool("consume", false, "Whether to run in rabbitmq consumer mode")
	workerIDPtr := flag.String("worker_id", "", "Worker ID (required by jobserver)")
	jobServerPtr := flag.String("jobserver", "", "Worker ID (required by jobserver)")

	flag.Parse()
	cliArgs = &types.CliArgs{
		Env:            *env,
		Pprof:          *prof,
		IsJobMode:      *job,
		IsRestMode:     *rest,
		IsTestMode:     *test,
		IsTestFileMode: *testFile,
		IsConsumeMode:  *consume,
		JobType:        *jobType,
		Url:            *url,
		Filename:       *filename,
		WorkerID:       *workerIDPtr,
		JobServerURL:   *jobServerPtr,
	}
	return cliArgs
}

// LoadConfig will appropriate config based on env
func LoadConfig(cliArgs *types.CliArgs) (appC *types.Config, err error) {
	env := cliArgs.Env
	log.Printf("CONFIG_LOAD: Loading %s configuration", env)
	configFile := fmt.Sprintf("config/%s.json", env)

	// 1. Read config file based on env
	file, err := os.Open(configFile)
	if err != nil {
		fmt.Printf("Failed to read config file: %v", err)
		return appC, err
	}

	// 2. Load config data accordingly
	var configData types.ConfigData
	data := json.NewDecoder(file)
	err = data.Decode(&configData)
	if err != nil {
		log.Printf("Failed to decode config data: %s\n", err)
		return appC, err
	}
	configData.Env = env
	configData.Args = cliArgs

	if cliArgs.JobServerURL != "" {
		configData.JobServer = cliArgs.JobServerURL
		log.Printf("CONFIG_LOAD: JobServerURL: %s\n", configData.JobServer)
	}

	if os.Getenv("JOBSERVER_ADDR") != "" {
		configData.JobServer = os.Getenv("JOBSERVER_ADDR")
		log.Printf("ENV_LOAD: JobServerURL: %s\n", configData.JobServer)
	}

	// cutils.PrintJson(configData)

	// 3. Create a s3 client to read cache
	s3ClientOptions := &s3Cache.ClientOpts{
		BucketName:        "sem3-html-cache-us-east",
		GetCredsFromVault: true,
		VaultKey:          "engineering/s3/crawl-user",
	}
	s3Client, err := s3Cache.MakeS3Client(s3ClientOptions)
	if err != nil {
		return appC, fmt.Errorf("Creating s3 client failed with error %s", err)
	}

	// 4. Create and rpc client for extraction service

	// 4.1 If deployment has any custom extraction timeout
	var extractionTimeout int = 60
	if os.Getenv("EXTRACTION_TIMEOUT") != "" {
		extractionTimeout, err = strconv.Atoi(os.Getenv("EXTRACTION_TIMEOUT"))
		if err != nil {
			log.Printf("Reading extraction timeout failed with error: %v\n", err)
		}
	}

	// 4.2 Create rpc client
	rpcClient, err := createRPCClient(os.Getenv("EXTRACTION_QUEUE"), extractionTimeout)
	if err != nil {
		return appC, fmt.Errorf("Creating extraction rpc client failed with error %s", err)
	}

	// 5. Create and unsupervised rpc client for extraction service
	var unsupervisedRPCClient *s3rpc.RPCClient
	aiQueue := os.Getenv("EXTRACTION_AI_QUEUE")
	if aiQueue != "" {
		unsupervisedRPCClient, err = createRPCClient(aiQueue, 100)
		if err != nil {
			return appC, fmt.Errorf("Creating unsupervised extraction rpc client failed with error %s", err)
		}
	}

	// 6. Initialize stats manager to track crawl and product metrics
	var statsdClient *statsd.Client
	var statsManager types.StatsManager
	stats.InitializeStatsManagerClient(&statsManager)

	datadoghost := os.Getenv("GLOBAL_DATADOG_HOST")
	if datadoghost != "" {
		statsdClient, err = statsd.New(datadoghost)
		if err != nil {
			return appC, fmt.Errorf("Creating statsd client failed with error %s", err)
		}
	}

	// 7. Create etl pipeline publishers
	var rawEtlPublisher publish.Publisher
	var ppEtlPublisher publish.Publisher
	var onDemandCrawlPublisher publish.Publisher
	var onDemandDiscoveryPublisher publish.Publisher

	brokerURI := os.Getenv("RABBITMQ_URI")
	configData.RdMsgBroker = brokerURI

	if configData.RdMsgBroker != "" {
		// amqpURI := fmt.Sprintf("amqps://%s:%s@%s:%s", configData.RPCBroker.Username, os.Getenv("SUPERVISED_RPC_PASS"), os.Getenv("RD_MSG_BROKER_URI"), strconv.Itoa(configData.RPCBroker.Port))
		if configData.RawEtlQueue != "" {
			_, err = rawEtlPublisher.Connect(brokerURI, configData.RawEtlQueue, true)
			if err != nil {
				err = cutils.PrintErr("DBS_RMQERR", fmt.Sprintf("failed to connect to (%s, %s)", configData.RdMsgBroker, configData.RawEtlQueue), err)
				return appC, err
			}
		}
		if configData.PpEtlQueue != "" {
			_, err = ppEtlPublisher.Connect(brokerURI, configData.PpEtlQueue, true)
			if err != nil {
				err = cutils.PrintErr("DBS_RMQERR", fmt.Sprintf("failed to connect to (%s, %s)", configData.RdMsgBroker, configData.PpEtlQueue), err)
				return appC, err
			}
		}
		if configData.OnDemandDiscoveryQueue != "" {
			_, err = onDemandDiscoveryPublisher.Connect(brokerURI, configData.OnDemandDiscoveryQueue, true)
			if err != nil {
				err = cutils.PrintErr("DBS_RMQERR", fmt.Sprintf("failed to connect to (%s, %s)", configData.RdMsgBroker, configData.OnDemandDiscoveryQueue), err)
				return appC, err
			}
		}
		if configData.OnDemandCrawlQueue != "" {
			_, err = onDemandCrawlPublisher.Connect(brokerURI, configData.OnDemandCrawlQueue, true)
			if err != nil {
				err = cutils.PrintErr("DBS_RMQERR", fmt.Sprintf("failed to connect to (%s, %s)", configData.RdMsgBroker, configData.OnDemandCrawlQueue), err)
				return appC, err
			}
		}
	}

	// Construct workerID by fetching instance_id
	var workerID string
	if cliArgs.IsJobMode {
		if cliArgs.WorkerID == "" {
			return appC, fmt.Errorf("WORKER_ID_ERR: worker_id param is mandatory for job mode")
		}

		workerID = cliArgs.WorkerID
		workerID = strings.Replace(workerID, "POD_IP", os.Getenv("MY_POD_IP"), -1)
		log.Printf("CONFIG_LOAD: WorkerID generated for the worker is %s\n", workerID)
	}

	//NOTE: RDSTORE PG CONNECT

	appC = &types.Config{
		WorkerID:                   workerID,
		RPCClient:                  rpcClient,
		S3Client:                   s3Client,
		StatsdClient:               statsdClient,
		StatsManager:               &statsManager,
		RedisCrawl:                 redisutils.NewRedisPool(os.Getenv("REDIS_HOST_ADDR")),
		RedisRdstore:               redisutils.NewRedisPool(os.Getenv("REDIS_HOST_ADDR")),
		RawEtlPublisher:            &rawEtlPublisher,
		PpEtlPublisher:             &ppEtlPublisher,
		OnDemandCrawlPublisher:     &onDemandCrawlPublisher,
		OnDemandDiscoveryPublisher: &onDemandDiscoveryPublisher,
		ConfigData:                 &configData,
	}

	// This connection is available for only few clients
	// for eg - this is not relevant for wrapperqa
	if unsupervisedRPCClient != nil {
		appC.UnsupervisedRPCClient = unsupervisedRPCClient
	}

	// Listen for wrapper/sitedetails live updates on redis pubsub
	go listenWrapperPubSubChannels(os.Getenv("REDIS_HOST_ADDR"))

	// Connect to mongo-crawl only for discovery_crawl and wraperqa types
	mongoURI := os.Getenv("MONGO_URI")
	if cutils.StringInSlice(cliArgs.JobType, []string{"discovery_crawl", "wrapperqa"}) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		mongoCrawl, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
		// mongoCrawl, err := mongo.Connect(ctx, configData.MongoCrawl)
		if err != nil {
			err = cutils.PrintErr("DBS_MONGOCRAWLERR", fmt.Sprintf("failed to connect to (%s)", configData.MongoCrawl), err)
			return appC, err
		}
		appC.MongoCrawl = mongoCrawl
	}

	// Translation service specific configurations
	// Connect to skus
	if configData.PGSkus != nil {
		appC.PGRaw = pg.Connect(&pg.Options{
			User:     configData.PGSkus.User,
			Password: os.Getenv("PG_SKUS_PASS"),
			Database: configData.PGSkus.DB,
			Addr:     os.Getenv("PG_SKUS_ADDR"),
			PoolSize: configData.PGSkus.PoolSize,
		})
	}
	// Create and translate rpc client for extraction service
	if configData.TranslateRPCConfig != nil {
		translateRPCClient, err := createRPCClient(os.Getenv("TRANSLATE_QUEUE"), 60)
		if err != nil {
			return appC, fmt.Errorf("Creating translate rpc client failed with error %s", err)
		}
		appC.TranslateRPCClient = translateRPCClient
	}

	validatelib.CompileSchema("SKUS", "")
	appC.ConfigData.RestRdstoreUpdate = os.Getenv("REST_RDSTOREUPDATE_ADDR")
	appC.ConfigData.ProxyRouter = os.Getenv("PROXY_ROUTER_ADDR")
	appC.ConfigData.CacheService = os.Getenv("CACHE_SERVICE_ADDR")
	appC.ConfigData.SitesDB = os.Getenv("SITESDB_SERVICE_ADDR")
	appC.ConfigData.Influx.Server = os.Getenv("INFLUXDB_ADDR")
	appC.ConfigData.WrapperServiceURI = os.Getenv("WRAPPER_SERVICE_URI")

	go stats.CollectStats(env, &statsManager, appC)
	return appC, nil
}

// Creates and returns a client instance to make RPC calls with
func createRPCClient(queueName string, timeout int) (*s3rpc.RPCClient, error) {
	var client s3rpc.RPCClient
	err := client.InitRPCClient(
		queueName,
		consume.ConsumerOptions{Timeout: timeout},
		os.Getenv("RABBITMQ_URI"),
		"",
	)
	if err != nil {
		return nil, nil
	}
	return &client, nil
}
