package data_test

import (
	"log"
	"strconv"
	"strings"
	"testing"

	"github.com/Semantics3/go-crawler/data"
	"github.com/Semantics3/go-crawler/types"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
	"github.com/Semantics3/sem3-go-data-consumer/consume"
	s3rpc "github.com/Semantics3/sem3-go-data-consumer/rpc"
	"github.com/go-pg/pg"

	"github.com/stretchr/testify/suite"
)

// Sitedetail test suite
type TranslateCrawlTestSuite struct {
	suite.Suite
	appC *types.Config
}

type buildCacheTestSuite struct {
	domain    string
	parentSKU string
	cache     map[string]data.TranslatedVal
}

type translateFieldsTestCase struct {
	domain        string
	parentSKU     string
	cacheBefore   int
	cacheAfter    int
	translateKeys []string
	rpcSwitch     int
}

// Initialize test suite
func (suite *TranslateCrawlTestSuite) SetupTest() {
	log.SetFlags(0)
	extractionRPCBroker := types.RPCBroker{
		Host:     "prod_rd_rabbitmq_spl-0.semantics3.com",
		Port:     5672,
		Username: "general",
		Password: "by4&jGAzII1XHhZU",
		Queue:    "rd-translate-prod",
	}

	var amqpURI = "amqp://" +
		extractionRPCBroker.Username + ":" + extractionRPCBroker.Password +
		"@" +
		extractionRPCBroker.Host + ":" + strconv.Itoa(extractionRPCBroker.Port)
	consumerOpts := consume.ConsumerOptions{Timeout: 60}

	var client s3rpc.RPCClient
	err := client.InitRPCClient(
		extractionRPCBroker.Queue,
		consumerOpts,
		amqpURI,
		"",
	)
	if err != nil {
		log.Printf("Error : %#v\n", err)
		return
	}

	//NOTE: RDSTORE PG CONNECT
	rdb := pg.Connect(&pg.Options{
		User:     "semantics3",
		Password: "semantics3",
		Database: "skus_production",
		Addr:     "skus-db.semantics3.com:5432",
		PoolSize: 10,
	})

	appC := &types.Config{
		TranslateRPCClient: &client,
		PGRaw:              rdb,
	}
	suite.appC = appC
}

func (suite *TranslateCrawlTestSuite) TestTranslateCrawl_BuildCache() {
	// Table based testing
	testCases := []*buildCacheTestSuite{
		&buildCacheTestSuite{domain: "test_translate.com", parentSKU: "sku1_cachesuccess", cache: map[string]data.TranslatedVal{"TURKISH_NAME1": data.TranslatedVal{Value: "ENGLISH_NAME1", Time: 1576054471}}},
		&buildCacheTestSuite{domain: "test_translate.com", parentSKU: "sku2_cachefail", cache: map[string]data.TranslatedVal{}},
	}
	tranObjName := "translation_metadata"
	tranKeys := []string{"name"}

	// Run through all test cases in table
	for i, t := range testCases {
		log.Printf("\nBUILDCACHE_TEST: Running test %d for URL (%s, %s)\n", i, t.domain, t.parentSKU)
		workflow := &types.CrawlWorkflow{
			DomainInfo: &ctypes.DomainInfo{
				ParentSku:  t.parentSKU,
				DomainName: t.domain,
			},
		}
		c, e := data.BuildTranslationCacheFromSkusDB(workflow, tranObjName, tranKeys, suite.appC)
		suite.Nil(e)
		suite.Equal(len(t.cache), len(c), "TRANSLATE_CACHE_FAILED: (%s, %s) expected %d results in cache, got %d", t.domain, t.parentSKU, len(t.cache), len(c))
	}
}

func (suite *TranslateCrawlTestSuite) TestTranslateCrawl_TranslateFieldsCacheSuccess() {

	tranObjName := "translation_metadata"

	// Table based testing
	testCases := []*translateFieldsTestCase{
		// Cache success [translation_metadata present for all keys required]
		&translateFieldsTestCase{domain: "test_translate.com", parentSKU: "sku1_cachesuccess", translateKeys: []string{"name"}, cacheBefore: 1, cacheAfter: 1, rpcSwitch: 1},
		// Cache success [translation_metadata present for all keys required], with switch off, should retain translated fields
		&translateFieldsTestCase{domain: "test_translate.com", parentSKU: "sku1_cachesuccess", translateKeys: []string{"name"}, cacheBefore: 1, cacheAfter: 1, rpcSwitch: 0},
		// Cache failed [translation_metadata present for all keys required], with switch off, should NOT HIT RPC
		&translateFieldsTestCase{domain: "test_translate.com", parentSKU: "sku2_cachefail", translateKeys: []string{"name"}, cacheBefore: 0, cacheAfter: 0, rpcSwitch: 0},
		// Cache failed [translation_metadata NOT present for all keys required]
		&translateFieldsTestCase{domain: "test_translate.com", parentSKU: "sku2_cachefail", translateKeys: []string{"name"}, cacheBefore: 0, cacheAfter: 2, rpcSwitch: 1},
		// Cache partially success [translation_metadata present for some keys required]
		&translateFieldsTestCase{domain: "test_translate.com", parentSKU: "sku3_cachefail_desc", translateKeys: []string{"name", "description"}, cacheBefore: 1, cacheAfter: 2, rpcSwitch: 1},
	}

	// Run through all test cases in table
	for i, t := range testCases {
		log.Printf("\nTRANSLATE_TEST: (testcase %d, domain %s, parentSKU %s)\n", i, t.domain, t.parentSKU)
		products := make([]map[string]interface{}, 0)
		sku := &data.Sku{Domain: t.domain, ParentSku: t.parentSKU}
		// To mimic crawled data [We do not want to crawl during the test]
		sku.Fetch(suite.appC.PGRaw, func(s *data.Sku) error {
			product := s.Data
			// Convert database record to crawled data equivalent
			to, tpresent := cutils.GetMapInterface(product, tranObjName)
			if tpresent {
				for origKey, origVal := range to {
					if strings.Contains(origKey, "_orig") {
						k := strings.Replace(origKey, "_orig", "", -1)
						product[k] = origVal
					}
				}
				delete(product, tranObjName)
			}
			products = append(products, s.Data)
			return nil
		})
		// jStr, _ := json.MarshalIndent(products, "", "  ")
		// log.Printf("JSON: %v\n", string(jStr))
		// os.Exit(1)

		// Will make a DB call and fill the cache
		workflow := &types.CrawlWorkflow{
			DomainInfo: &ctypes.DomainInfo{
				ParentSku:  sku.ParentSku,
				DomainName: sku.Domain,
			},
			Data: types.ExtractionResponse{
				Products: products,
			},
		}
		c, _ := data.BuildTranslationCacheFromSkusDB(workflow, tranObjName, t.translateKeys, suite.appC)
		cacheBefore := len(c)

		translationFlags := &ctypes.TranslationFlags{
			Source:    "tr",
			Target:    "en",
			RPCSwitch: t.rpcSwitch,
			Fields:    t.translateKeys,
			JobTypes:  map[string]int{"recrawl": 1},
		}
		e := data.TranslateFields(workflow, tranObjName, translationFlags, c, suite.appC)
		cacheAfter := len(c)

		suite.Nil(e)
		suite.Equal(t.cacheBefore, cacheBefore, "(%s, %s) Expected %d, got %d", t.domain, t.parentSKU, t.cacheBefore, cacheBefore)
		suite.Equal(t.cacheAfter, cacheAfter, "(%s, %s) Expected %d, got %d", t.domain, t.parentSKU, t.cacheAfter, cacheAfter)

		for _, p := range products {
			suite.NotNil(p[tranObjName])
		}
	}
}

//func (suite *TranslateCrawlTestSuite) TestTranslateCrawl_TranslateFieldsCacheFail() {

//sku := &data.Sku{Domain: "test_translate.com", ParentSku: "sku1_cachefail"}
//data := make([]map[string]interface{}, 0)

//sku.Fetch(suite.appC.PGRaw, func(s *data.Sku) error {
//data = append(data, s.Data)
//return nil
//})

//// Table based testing
//testCases := []*translateFieldsSuite{
//&translateFieldsSuite{
//products: data,
//cache:    make(map[string]data.TranslatedVal, 0),
//},
//}

//tranObjName := "translation_metadata"
//tranKeys := []string{"name"}

//// Run through all test cases in table
//for _, t := range testCases {
//e := data.TranslateFields(t.products, tranObjName, tranKeys, t.cache, suite.appC)

//suite.Nil(e)
//suite.Equal(len(t.cache), 1)

//for _, p := range t.products {
//suite.NotNil(p[tranObjName])
//}

//}
//}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestTranslateCrawlTestSuite(t *testing.T) {
	suite.Run(t, new(TranslateCrawlTestSuite))
}
