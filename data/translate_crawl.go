package data

import (
	"fmt"
	"log"
	"time"

	"github.com/go-pg/pg"

	"github.com/Semantics3/go-crawler/types"
	redisutils "github.com/Semantics3/sem3-go-crawl-utils/redis"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

func ShouldTranslateForJob(workflow *types.CrawlWorkflow, jobType string) (yes bool) {
	sitedetail := workflow.DomainInfo.Sitedetail
	// If sitedetail itself is not present, NO Translation
	if sitedetail == nil {
		return false
	}

	// If translationFlags field is not present, NO Translation
	// 99% of sites will exit here
	translationFlags := sitedetail.TranslationFlags
	if translationFlags == nil {
		return false
	}

	temp := make(map[string]interface{}, 0)
	for k, v := range translationFlags.JobTypes {
		if v == 1 {
			temp[k] = true
		} else {
			temp[k] = false
		}
	}

	// In the rare case that translationFlags is present, we check
	// current jobType has to translate
	shouldRecrawlTranslate, ok := cutils.GetBoolKey(temp, jobType)
	if ok && shouldRecrawlTranslate {
		return true
	}
	return false
}

func ApplyTranslation(workflow *types.CrawlWorkflow, appC *types.Config) error {
	workflow.IsTranslateCrawl = true

	translationFlags := workflow.DomainInfo.Sitedetail.TranslationFlags
	translateKeys := translationFlags.Fields

	translationObjName := "translation_metadata"

	// Build cache from SkusDB for fields of current product
	cache, _ := BuildTranslationCacheFromSkusDB(workflow, translationObjName, translateKeys, appC)

	// Translate value of new fields or new product variations
	return TranslateFields(workflow, translationObjName, translationFlags, cache, appC)
}

// BuildTranslationCacheFromSkusDB reads product data from skus-db
// and picks up previously translated values for name, desc etc
func BuildTranslationCacheFromSkusDB(workflow *types.CrawlWorkflow,
	translationObjName string, translateKeys []string,
	appC *types.Config) (cache map[string]TranslatedVal, err error) {

	domain := workflow.DomainInfo.DomainName
	parentSKU := workflow.DomainInfo.ParentSku
	// CACHE_START: Building cache to avoid re transaltion.
	// Caches all name, desc original values to -> translated (english) values
	cache = make(map[string]TranslatedVal, 0)
	sku := &Sku{
		Domain:    domain,
		ParentSku: parentSKU,
	}
	workflow.RawData = make(map[string]map[string]interface{}, 0)

	sku.Fetch(appC.PGRaw, func(s *Sku) error {
		translationMeta, ok := cutils.GetMapInterface(s.Data, translationObjName)
		if ok {
			for _, k := range translateKeys {
				origKey := fmt.Sprintf("%s_orig", k)
				origEpoch := fmt.Sprintf("%s_epoch", k)
				origVal, ok := cutils.GetStringKey(translationMeta, origKey)
				origEpochVal, _ := cutils.GetInt64Key(translationMeta, origEpoch)
				if ok {
					translatedText, ok := cutils.GetStringKey(s.Data, k)
					if ok {
						cache[origVal] = TranslatedVal{Value: translatedText, Time: origEpochVal}

						// If name or description is present
						childSku, _ := cutils.GetStringKey(s.Data, "sku")
						childSkuTranslatedFields, ok := workflow.RawData[childSku]
						if !ok {
							childSkuTranslatedFields = map[string]interface{}{}
						}
						childSkuTranslatedFields[k] = translatedText
						workflow.RawData[childSku] = childSkuTranslatedFields
					}
				}
			}
		}
		return nil
	})
	// CACHE_END: Cache building done.

	return cache, nil
}

// TranslateFields translates text fields for all variations passed in using cache passed in
// This function also updates the cache as it translates new field values not seen before
func TranslateFields(workflow *types.CrawlWorkflow,
	translationObjName string, translationFlags *ctypes.TranslationFlags,
	cache map[string]TranslatedVal, appC *types.Config) (err error) {

	products := workflow.Data.Products
	translateKeys := translationFlags.Fields
	src := translationFlags.Source
	target := translationFlags.Target

	productUrl := ""
	childSku := ""

	for childSKUNum, product := range products {

		productUrl, _ = cutils.GetStringKey(product, "url")
		childSku, _ = cutils.GetStringKey(product, "sku")

		if product[translationObjName] == nil {
			product[translationObjName] = make(map[string]interface{}, 0)
		}
		to, _ := cutils.GetMapInterface(product, translationObjName)

		for _, k := range translateKeys {

			origKey := fmt.Sprintf("%s_orig", k)
			origEpoch := fmt.Sprintf("%s_epoch", k)
			origVal, ok := cutils.GetStringKey(product, k)

			// If name, desc, color or the translation key has a value in the product data
			if ok {
				translatedVal, cok := cache[origVal]
				if cok {
					log.Printf("TRANSLATE_CACHE_SUCCESS: (url %s, child_sku %s, key %s, epoch %d, orig %s, en %s)\n",
						productUrl, childSku, k, translatedVal.Time, origVal, translatedVal.Value)
				} else {
					if translationFlags.RPCSwitch != 1 {
						log.Printf("TRANSLATE_CACHE_MISS_SWITCH_OFF: (url %s, child_sku %s, key %s), so skipping translate RPC call\n", productUrl, childSku, k)
						// If this child_sku was previously translated, use the translated name or
						// description
						childSkuTranslatedFields, ok := workflow.RawData[childSku]
						if ok {
							translatedText, ok := cutils.GetStringKey(childSkuTranslatedFields, k)
							if ok {
								product[k] = translatedText
							}
						}
						continue
					}
					log.Printf("TRANSLATE_CACHE_MISS: (url %s, child_sku %s, key %s, orig %s)\n", productUrl, childSku, k, origVal)
					if workflow.JobInput != nil {
						err = UpdateTranslationStatsInRedis(workflow.JobInput.JobID, childSku, k, childSKUNum, appC)
						if err != nil {
							log.Printf("%v\n", err)
						}
					}
					translatedText := ""
					translatedText, err = TranslateRPC(origVal, src, target, appC)
					if err != nil {
						msg := fmt.Sprintf("TRANSLATE_KEY_ERR: (url %s, child_sku %s, key %s, orig %s, err %v)\n", productUrl, childSku, k, origVal, err)
						log.Printf(msg)
						return fmt.Errorf(msg)
					}

					translatedVal = TranslatedVal{Value: translatedText, Time: time.Now().Unix()}
					log.Printf("TRANSLATE_RPC_RESP: (url %s, child_sku %s, key %s, orig %s, en %s)\n", productUrl, childSku, k, origVal, translatedText)

					// Cache the translation value for other variations
					cache[origVal] = translatedVal
				}

				// Copy the original Turkish/other language value inside tranlation_meta
				// object
				to[origKey] = origVal
				to[origEpoch] = translatedVal.Time

				// SET_PRODUCT_DATA: Set name, desc in product data to english value
				// translatedVal.Value below comes from either cache or Translate() func call
				product[k] = translatedVal.Value

			} else {
				// name or description field has no data from wrapper extraction
				// In this case, nothing to translate
				log.Printf("TRANSLATE_KEY_NO_DATA: (url %s, child_sku %s, key %s, orig %s)\n", productUrl, childSku, k, origVal)
			}
		}

		// SET_PRDOUCT_TRANSLATIONMETA
		product[translationObjName] = to
	}

	return nil
}

// TranslateRPC performs RPC call to translate service and gets english text
func TranslateRPC(key, src, targetlang string, appC *types.Config) (string, error) {

	method := "translate"

	r, err := appC.TranslateRPCClient.Call(method, key, targetlang)
	if err != nil {
		log.Printf("TRANSLATE_KEY_RPC_ERROR: Method: %s, Target Language: %s, Key: %s, Error: %#v\n", method, targetlang, key, err)
		return "", err
	}

	log.Printf("RPC_RESPONSE: %s\n", r)
	resp, rok := r.(map[string]interface{})
	if !rok {
		log.Printf("TRANSLATE_KEY_RPC_RESP_NIL_ERROR: Method: %s, Target Language: %s, Key: %s\n", method, targetlang, key)
		return "", fmt.Errorf("Nil Response payload")
	}

	resp, rok = cutils.GetMapInterface(resp, "result")
	if !rok {
		log.Printf("TRANSLATE_KEY_RPC_RESP_NIL_ERROR: Method: %s, Target Language: %s, Key: %s\n", method, targetlang, key)
		return "", fmt.Errorf("Nil Response payload")
	}

	errr, eok := cutils.GetStringKey(resp, "error")
	if eok {
		log.Printf("TRANSLATE_KEY_RPC_RESP_ERROR: Method: %s, Target Language: %s, Key: %s, Error: %s\n", method, targetlang, key, errr)
		return "", fmt.Errorf("%s", errr)
	}

	text, tok := cutils.GetStringKey(resp, "text")
	if !tok {
		log.Printf("TRANSLATE_KEY_RPC_RESP_TEXT_ERROR: Method: %s, Target Language: %s, Key: %s\n", method, targetlang, key)
		return "", fmt.Errorf("Nil Text")
	}

	return text, nil
}

//NOTE: Type Representing skus db record.
type Sku struct {
	tableName struct{}               `sql:"skus"`
	ID        int64                  `sql:"id"`
	Domain    string                 `sql:"domain"`
	ParentSku string                 `sql:"parent_sku"`
	Data      map[string]interface{} `sql:"data"`
}

type TranslatedVal struct {
	Value string `json:"value"`
	Time  int64  `json:"time"`
}

func (s *Sku) Fetch(db *pg.DB, cb func(s *Sku) error) error {

	err := db.
		Model((*Sku)(nil)).
		Where("domain = ?", s.Domain).
		Where("parent_sku = ?", s.ParentSku).
		ForEach(cb)

	if err != nil {
		log.Printf("TRANSLATE_RECRAWL_FETCH_ERROR: %#v\n", err)
		return err
	}

	return nil
}

// UpdateTranslationStatsInRedis - Update translation stats for job
func UpdateTranslationStatsInRedis(jobID string, childSKU string, field string, childSKUNum int, appC *types.Config) (err error) {
	if appC == nil || appC.RedisCrawl == nil {
		return fmt.Errorf("UPDATE_TRANSLATION_REDIS_ERR: Redis pool nil")
	}
	hashName := fmt.Sprintf("job_translation_stats;%s", jobID)
	_, err = redisutils.HIncrby(appC.RedisCrawl, hashName, "total_calls", 1)
	if err != nil {
		return fmt.Errorf("UPDATE_TRANSLATION_TOTAL_ERR: %v", err)
	}
	if childSKUNum == 0 {
		_, err = redisutils.HIncrby(appC.RedisCrawl, hashName, fmt.Sprintf("parent_sku_%s_calls", field), 1)
		if err != nil {
			return fmt.Errorf("UPDATE_TRANSLATION_PARENTSTATS_ERR: %v", err)
		}
	}
	_, err = redisutils.HIncrby(appC.RedisCrawl, hashName, fmt.Sprintf("child_sku_%s_calls", field), 1)
	if err != nil {
		return fmt.Errorf("UPDATE_TRANSLATION_CHILDSTATS_ERR: %v", err)
	}
	return nil
}
