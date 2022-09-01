package amazon

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Semantics3/go-crawler/sources"
	"github.com/Semantics3/go-crawler/sources/amazon/paapi5"
	"github.com/Semantics3/go-crawler/sources/amazon/paapi5/types"
	ctypes "github.com/Semantics3/go-crawler/types"
)

var amazonHandle *Handle

func init() {
	amazonHandle = NewAmazonActor()
}

type AmazonRequest struct {
	URL     string
	JobType string
	Time    time.Time
	Type    types.Operation
	asin    string
	appC    *ctypes.Config
	respCh  chan<- *types.ItemResponse
	// retry   int
}

type BatchRequest struct {
	Requests []AmazonRequest
	JobType  string
	Type     types.Operation
	locale   types.Locale
	retry    int
	appC     *ctypes.Config
}

type actor struct {
	err        error
	errCode    string
	client     *paapi5.Client
	reqQueue   <-chan AmazonRequest
	batchQueue chan *BatchRequest
	quitActor  <-chan bool
	maxRetry   int
}

// Handle - Handle to communicate updates with jobtype Actor/Listener
type Handle struct {
	tx   chan<- AmazonRequest
	quit chan<- bool
}

func (h *Handle) requestActor(req AmazonRequest) (resp *types.ItemResponse) {
	defer func() {
		if r := recover(); r != nil {
			resp.Error = fmt.Errorf("HANDLE_SEND_ERR: Sending message failed with error %v", r)
		}
	}()
	respCh := make(chan *types.ItemResponse)
	req.respCh = respCh
	h.tx <- req
	resp = <-respCh

	if resp.Error != nil {
		log.Printf("HANDLE_FATAL_ERR: %v\n", resp.Error)
	}
	return resp
}

func (h *Handle) GetItems(appC *ctypes.Config, url string, jobType string) ([]map[string]interface{}, string, error) {
	req := AmazonRequest{
		URL:     url,
		JobType: jobType,
		Type:    types.GetItems,
		Time:    time.Now(),
		appC:    appC,
	}
	resp := h.requestActor(req)
	return resp.Data, resp.Code, resp.Error
}

func (h *Handle) GetVariations(appC *ctypes.Config, url string, jobType string) ([]map[string]interface{}, string, error) {
	req := AmazonRequest{
		URL:     url,
		JobType: jobType,
		Type:    types.GetVariations,
		Time:    time.Now(),
		appC:    appC,
	}
	resp := h.requestActor(req)
	return resp.Data, resp.Code, resp.Error
}

// Delete - Close the actor
func (h *Handle) Delete() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("HANDLE_SEND_ERR: %v", r)
		}
	}()
	h.quit <- true
	return
}

func NewAmazonActor() (h *Handle) {
	channel := make(chan AmazonRequest, 100)
	batch := make(chan *BatchRequest, 10)
	quitCh := make(chan bool)
	client, code, err := amazonClient()
	w := &actor{
		quitActor:  quitCh,
		reqQueue:   channel,
		batchQueue: batch,
		client:     client,
		err:        err,
		errCode:    code,
		maxRetry:   10,
	}
	go w.requestToBatch(10, time.Second)
	go w.spawn()

	return &Handle{
		tx:   channel,
		quit: quitCh,
	}
}

func amazonClient() (client *paapi5.Client, code string, err error) {
	return paapi5.NewClient()
}

func (a *actor) spawn() {
	log.Printf("AMAZON_ACTOR_BEGIN\n")
MAIN_LOOP:
	for {
		select {
		case batch := <-a.batchQueue:
			res := a.getResults(batch)
			log.Println("AMAZON_BATCH_TYPE:", batch.Type, "AMAZON_BATCH_LENGTH:", len(batch.Requests))
			for idx, req := range batch.Requests {
				req.respCh <- res[idx]
			}
		case <-a.quitActor:
			log.Printf("AMAZON_BATCH_END: quit signal recieved\n")
			break MAIN_LOOP
		}
	}
}

func (a *actor) getResults(batch *BatchRequest) []*types.ItemResponse {
	source := "amazon"
	if batch.JobType == "realtimeapi" {
		source = "api_amazon"
	}
	check, err := sources.CheckRateLimitPerSecond(batch.appC.RedisRdstore, source)
	if err != nil {
		err = fmt.Errorf("AMAZON_RATELIMIT_ERR: %v", err)
		return types.NewMultiItemResponse(len(batch.Requests), "AMAZON_RATELIMIT_ERR", err)
	}

	if !check {
		if batch.retry >= a.maxRetry {
			err = fmt.Errorf("too many requests, %d retries failed", batch.retry)
			return types.NewMultiItemResponse(len(batch.Requests), "AMAZON_RATELIMIT_EXCEEDED", err)
		}
		// calculate sleep time
		milliEpoch := time.Now().UnixNano() / int64(time.Millisecond)
		secEpoch := time.Now().Unix()
		sleepTime := 1000 - (milliEpoch - secEpoch*1000)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		batch.retry++
		return a.getResults(batch)
	}

	var products []map[string]interface{}
	var code string

	if batch.Type == types.GetItems {
		req := batch.Requests
		asins := make([]string, len(req))
		for idx, val := range req {
			asins[idx] = val.asin
		}
		return a.client.GetItemsFromAsins(context.TODO(), batch.locale, asins)
	}

	products, code, err = a.client.GetVariationsFromURL(context.TODO(), batch.Requests[0].URL)
	res := types.NewItemResponse(products, code, err)
	return []*types.ItemResponse{res}
}

func (a *actor) requestToBatch(maxItems int, maxWaitTime time.Duration) {
	localeBatchMap := map[types.Locale]*BatchRequest{}
	ticker := time.NewTicker(maxWaitTime)
MAIN_LOOP:
	for {
		select {
		case req := <-a.reqQueue:
			locale, asin, code, err := paapi5.GetLocaleAsinFromURL(req.URL)
			if err != nil {
				req.respCh <- types.NewItemResponse(nil, code, err)
			} else if req.Type == types.GetVariations {
				batch := &BatchRequest{
					Requests: []AmazonRequest{req},
					JobType:  req.JobType,
					Type:     req.Type,
					locale:   locale,
					appC:     req.appC,
				}
				a.batchQueue <- batch
			} else {
				req.asin = asin
				batch, ok := localeBatchMap[locale]
				if batch == nil || !ok {
					batch = &BatchRequest{
						Requests: []AmazonRequest{},
						JobType:  req.JobType,
						Type:     req.Type,
						locale:   locale,
						appC:     req.appC,
					}
				}

				batch.Requests = append(batch.Requests, req)
				localeBatchMap[locale] = batch
				if len(batch.Requests) >= maxItems {
					a.batchQueue <- batch
					localeBatchMap[locale] = nil
				}
			}
		case <-ticker.C:
			for locale, val := range localeBatchMap {
				if val != nil && len(val.Requests) > 0 {
					a.batchQueue <- val
					localeBatchMap[locale] = nil
				}
			}
			// log.Println("Tick at", t)
		case <-a.quitActor:
			log.Printf("AMAZON_ACTOR_END: quit signal recieved\n")
			break MAIN_LOOP
		}
	}
}
