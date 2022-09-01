package m101

import (
	"fmt"
	"log"
	"time"

	"github.com/Semantics3/go-crawler/sources"
	ctypes "github.com/Semantics3/go-crawler/types"
)

var m101Handle *Handle

func init() {
	m101Handle = m101Actor()
}

type M101Request struct {
	URL    string
	Time   time.Time
	appC   *ctypes.Config
	respCh chan<- Resp
	retry  int
}

// Resp - Resp type
type Resp struct {
	Data    []map[string]interface{}
	Err     error
	ErrCode string
}

type actor struct {
	err       error
	errCode   string
	reqQueue  <-chan M101Request
	client    *M101Client
	quitActor <-chan bool
	maxRetry  int
}

// Handle - Handle to communicate updates with jobtype Actor/Listener
type Handle struct {
	tx   chan<- M101Request
	quit chan<- bool
}

func (h *Handle) requestActor(req M101Request) (resp Resp) {
	defer func() {
		if r := recover(); r != nil {
			resp.Err = fmt.Errorf("HANDLE_SEND_ERR: Sending message failed with error %v", r)
		}
	}()
	respCh := make(chan Resp)
	req.respCh = respCh
	h.tx <- req
	resp = <-respCh

	if resp.Err != nil {
		log.Printf("HANDLE_FATAL_ERR: %v\n", resp.Err)
	}
	return resp
}

func (h *Handle) GetResults(appC *ctypes.Config, url string) ([]map[string]interface{}, string, error) {
	req := M101Request{
		URL:  url,
		Time: time.Now(),
		appC: appC,
	}
	resp := h.requestActor(req)
	return resp.Data, resp.ErrCode, resp.Err
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

func m101Actor() (h *Handle) {
	channel := make(chan M101Request, 25)
	quitCh := make(chan bool)
	client, code, err := NewM101Client()
	w := &actor{
		quitActor: quitCh,
		reqQueue:  channel,
		client:    client,
		err:       err,
		errCode:   code,
		maxRetry:  10,
	}
	go w.spawn()

	return &Handle{
		tx:   channel,
		quit: quitCh,
	}
}

func (a *actor) spawn() {
	log.Printf("M101_ACTOR_BEGIN\n")
MAIN_LOOP:
	for {
		select {
		case req := <-a.reqQueue:
			req.respCh <- a.getResults(req)
		case <-a.quitActor:
			log.Printf("M101_ACTOR_END: quit signal recieved\n")
			break MAIN_LOOP
		}
	}
}

func (a *actor) getResults(req M101Request) Resp {
	if a.err != nil {
		return Resp{Err: a.err, ErrCode: a.errCode}
	}
	check, err := sources.CheckRateLimitPerSecond(req.appC.RedisRdstore, "m101")
	if err != nil {
		err = fmt.Errorf("M101_RATELIMIT_ERR: %v", err)
		return Resp{
			ErrCode: "M101_RATELIMIT_ERR",
			Err:     err,
		}
	}
	if !check {
		if req.retry >= a.maxRetry {
			return Resp{
				ErrCode: "M101_RATELIMIT_EXCEEDED",
				Err:     fmt.Errorf("too many requests, %d retries failed", req.retry),
			}
		}
		// calculate sleep time
		milliEpoch := time.Now().UnixNano() / int64(time.Millisecond)
		secEpoch := time.Now().Unix()
		sleepTime := 1000 - (milliEpoch - secEpoch*1000)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		return a.getResults(req)
	}

	// offerID, err := a.client.GetOfferID(req.URL)
	// if err != nil {
	// 	return Resp{
	// 		Err:     err,
	// 		ErrCode: "M101_OFFERID_REQUEST_ERR",
	// 	}
	// }
	// res, err := a.client.GetOfferIDItem(offerID)
	// if err != nil {
	// 	return Resp{
	// 		Err:     err,
	// 		ErrCode: "M101_OFFERID_ITEM_REQUEST_ERR",
	// 	}
	// }

	// As M101 changed their querying mechanism, we can get all data with just one API call instead of two
	res, err := a.client.GetProductData(req.URL)
	if err != nil {
		return Resp{
			Err:     err,
			ErrCode: "M101_API_REQUEST_ERR",
		}
	}
	products := []map[string]interface{}{}
	if res != nil {
		products = append(products, res)
	}
	return Resp{
		Data: products,
	}
}
