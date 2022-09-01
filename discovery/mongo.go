package discovery

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	jobutils "github.com/Semantics3/sem3-go-crawl-utils/jobs"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

// WriteCrawlDataToMongo will bulk insert products to mongo
func WriteCrawlDataToMongo(workflow *types.CrawlWorkflow, appC *types.Config) (err error) {

	// 1. Get database name
	site := workflow.DomainInfo.DomainName
	dbName := getDatabaseName(site)

	// 2. Get collection prefix
	var timeCreated int64
	jobConfig, err := jobutils.GetJobStatus(workflow.JobInput.JobID, appC.ConfigData.JobServer)
	if err != nil {
		return err
	}
	if ds, ok := cutils.GetIntKey(jobConfig.JobParams, "daily_sets"); ok && ds == 1 {
		// For microsoft custom discovery jobs, data has to be written
		// to new collection everyday baded on midnight timestamp
		timeCreated = getMidnightTimestamp() * 1000
		log.Printf("Identified the job as daily-sets. Collecting %d as timestamp for mongo collection", timeCreated)
	} else {
		timeCreated = jobConfig.State.TimeCreated
	}
	// collPrefix, err := getCollNamePrefix(timeCreated, workflow.JobInput.JobID)
	collPrefix := fmt.Sprintf("%s_%d", strings.ReplaceAll(workflow.JobInput.JobID, "-", "_"), (timeCreated / 1000))

	// 3. Insert products/categories to mongo
	err = insertRecordsToDatabase(dbName, collPrefix, workflow, appC)
	return
}

func getDatabaseName(site string) (dbName string) {
	dbName = fmt.Sprintf("crawl_%s", site)
	dbName = strings.Replace(dbName, ".", "_", -1)
	return
}

func getCollNamePrefix(timestamp int64, jobID string) (collPrefix string, err error) {
	timeStarted := timestamp / 1000
	id, err := utils.Md5Hash(jobID)
	if err != nil {
		return "", cutils.PrintErr("CREATING_MD5HASH_FAILED", fmt.Sprintf("Failed to create md5hash for %s while identifying collection name", jobID), err)
	}
	collPrefix = fmt.Sprintf("c_%s_1_%d", id, timeStarted)
	return
}

func getMidnightTimestamp() int64 {
	current := time.Now()
	midnight := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, time.Local)
	return midnight.Unix()
}

func insertRecordsToDatabase(db string, collPrefix string, workflow *types.CrawlWorkflow, appC *types.Config) (err error) {
	// TODO: Should we consider links & cart aswell ?
	dataKeys := []string{"products", "category"}
	for _, key := range dataKeys {
		records := make([]map[string]interface{}, 0)
		switch key {
		case "products":
			for _, v := range workflow.Data.Products {
				records = append(records, v)
			}
		case "category":
			for _, v := range workflow.Data.Categories {
				records = append(records, v)
			}
		}

		// Skip if key has no items
		l := len(records)
		if l == 0 {
			continue
		}
		insert(key, db, collPrefix, records, appC)
	}
	return
}

func insert(key string, db string, collPrefix string, records []map[string]interface{}, appC *types.Config) {
	success, failures := 0, 0
	collName := fmt.Sprintf("%s_%s", collPrefix, key)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	collection := appC.MongoCrawl.Database(db).Collection(collName)
	opts := options.Update().SetUpsert(true)

	for _, record := range records {
		// _, err := collection.InsertOne(ctx, record)
		val, ok := cutils.GetStringKey(record, "_id")
		if !ok {
			log.Printf("Finding _id in the record failed: %v\n", record)
			failures++
			continue
		}

		filter := bson.M{"_id": val}
		update := bson.M{
			"$set": record,
		}
		_, err := collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			log.Printf("Inserting record failed with error: %v\n", err)
			failures++
			continue
		}
		success++
	}
	log.Printf("MONGO_BATCH_STATS: DB: %s, COLL: %s, KEY: %s, SUCCESS: %d/%d, FAILURES: %d/%d\n", db, collName, key, success, len(records), failures, len(records))
}
