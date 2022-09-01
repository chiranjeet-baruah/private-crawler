package types

import (
	"github.com/DataDog/datadog-go/statsd"
	"github.com/Jeffail/tunny"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	s3Cache "github.com/Semantics3/sem3-go-crawl-utils/webcache/s3"
	publish "github.com/Semantics3/sem3-go-data-consumer/publish"
	s3rpc "github.com/Semantics3/sem3-go-data-consumer/rpc"
	pg "github.com/go-pg/pg"
	"github.com/gomodule/redigo/redis"
	mongo "go.mongodb.org/mongo-driver/mongo"
)

type (
	CliArgs struct {
		Env            string `json:"env"`
		Pprof          bool   `json:"pprof"`
		IsJobMode      bool   `json:"job"`
		IsRestMode     bool   `json:"rest"`
		IsTestMode     bool   `json:"test"`
		IsTestFileMode bool   `json:"test-file"`
		IsConsumeMode  bool   `json:"consume"`
		JobType        string `json:"test-job-type"`
		Url            string `json:"url"`
		Filename       string `json:"file"`
		WorkerID       string `json:"worker_id"`
		JobServerURL   string `json:"jobserver"`
	}

	ConfigData struct {
		Args                       *CliArgs                     `json:"cli_args"`
		Env                        string                       `json:"env"`
		RedisCrawl                 string                       `json:"redis_crawl"`
		RedisRecrawlDiscovery      string                       `json:"redis_recrawldiscovery"`
		RestRdstoreUpdate          string                       `json:"rest_rdstoreupdate"`
		RealtimeDataForwarderQueue string                       `json:"realtime_data_forwarder_queue"`
		MongoCrawl                 string                       `json:"mongo_crawl"`
		ProxyRouter                string                       `json:"proxy_router"`
		CacheService               string                       `json:"cache_service"`
		Influx                     InfluxConfig                 `json:"influx"`
		RPCBroker                  RPCBroker                    `json:"extraction_rpc"`
		UnsupervisedRPCBroker      RPCBroker                    `json:"unsupervised_extraction_rpc"`
		TranslateRPCConfig         *RPCBroker                   `json:"translation_rpc"`
		RawEtlQueue                string                       `json:"raw_etl_queue"`
		PpEtlQueue                 string                       `json:"pp_etl_queue"`
		OnDemandCrawlQueue         string                       `json:"ondemand_crawl_queue"`
		OnDemandDiscoveryQueue     string                       `json:"ondemand_discovery_queue"`
		RdMsgBroker                string                       `json:"rd_msg_broker"`
		JobServer                  string                       `json:"job_server"`
		SitesDB                    string                       `json:"sitesdb"`
		WrapperServiceURI          string                       `json:"wrapper_service_uri"`
		RdEtlController            string                       `json:"rd_etl_controller"`
		NotifierService            string                       `json:"notifier_service"`
		ConsumeQueueConfig         []QueueConfig                `json:"consume_queues"`
		PublishQueueConfig         []QueueConfig                `json:"publish_queues"`
		ConsumerSitePoolConfig     map[string]int               `json:"consume_site_pool_map"`
		PGSkus                     *PGSkus                      `json:"pg_skus"`
		SourceConfig               map[string]map[string]string `json:"source_config"`
	}

	Config struct {
		WorkerID                       string `json:"worker_id"`
		JobParams                      *ctypes.CrawlJobParams
		RedisCrawl                     *redis.Pool
		RedisRdstore                   *redis.Pool
		MongoCrawl                     *mongo.Client
		S3Client                       *s3Cache.Client
		RPCClient                      *s3rpc.RPCClient
		UnsupervisedRPCClient          *s3rpc.RPCClient
		StatsdClient                   *statsd.Client
		StatsManager                   *StatsManager
		RawEtlPublisher                *publish.Publisher
		PpEtlPublisher                 *publish.Publisher
		OnDemandCrawlPublisher         *publish.Publisher
		OnDemandDiscoveryPublisher     *publish.Publisher
		ConfigData                     *ConfigData
		RealtimeDataForwarderPublisher *publish.Publisher
		Publishers                     *PublisherConfig       `json:"publish_queues"`
		ConsumerSitePoolMap            map[string]*tunny.Pool `json:"consumer_site_pool_map"`
		PGRaw                          *pg.DB                 //NOTE: Skus db connection.
		TranslateRPCClient             *s3rpc.RPCClient
	}

	PGSkus struct {
		User     string `json:"user"`
		Password string `json:"password"`
		Addr     string `json:"addr"`
		DB       string `json:"db"`
		PoolSize int    `json:"pool_size"`
	}

	InfluxConfig struct {
		Server            string `json:"server"`
		Database          string `json:"database"`
		CrawlMetrics      string `json:"crawl_metrics"`
		ProductMetrics    string `json:"product_metrics"`
		ExtractionMetrics string `json:"extraction_metrics"`
		Protocol          string `json:"protocol"`
	}

	RPCBroker struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"user"`
		Password string `json:"pass"`
		Queue    string `json:"queue"`
		Timeout  *int   `json:"timeout"`
	}

	QueueConfig struct {
		PrefetchCount int    `json:"prefetch_count"`
		QueueName     string `json:"queue_name"`
		ServiceName   string `json:"service_name"`
	}

	PublisherConfig struct {
		CrawlWorkerPublisher *publish.Publisher `json:"crawl_worker_publisher"`
	}
)
