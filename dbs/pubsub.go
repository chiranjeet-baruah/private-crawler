package dbs

import (
	"encoding/json"
	"log"
	"time"

	"github.com/Semantics3/sem3-go-crawl-utils/sitedetails"
	"github.com/gomodule/redigo/redis"
)

// LiveUpdateRequest will define type for wrapper/sitedetails realtime updates using redis pubsub
type LiveUpdateRequest struct {
	Site string `json:"site"`
	ID   string `json:"id"`
}

const (
	// this is the only way that we can possibly know that the TCP connection is dead
	readCommandTimeout  = 45 * time.Second
	writeCommandTimeout = 45 * time.Second
	connectTimeout      = 30 * time.Second
	retryTimeout        = 10 * time.Second
	// sends a PING to the server over the pubsub connection. expects a PONG or an error in response.
	healthCheckPeriod = 30 * time.Second
)

func onReceivePubSubMessage(channel string, data []byte) error {
	liveRequest := LiveUpdateRequest{}
	if err := json.Unmarshal(data, &liveRequest); err != nil {
		log.Printf("REDIS_PUBSUB: Unmarshalling message: %s failed with error: %v\n", string(data), err)
		return nil
	}
	if liveRequest.ID == "" || liveRequest.Site == "" {
		log.Printf("REDIS_PUBSUB: Skipping, Missing id or site in message: %s\n", string(data))
		return nil
	}
	log.Println("REDIS_PUBSUB: Message received: ", string(data))
	if channel == "sitedetail_live_updates" {
		sitedetails.RemoveSitedetailFromCache(liveRequest.Site)
	} else if channel == "wrapper_live_updates" {
		sitedetails.RemoveWrapperFromCache(liveRequest.ID, liveRequest.Site)
	} else {
		log.Printf("REDIS_PUBSUB: Unknown channel: %s, message: %s\n", channel, data)
	}
	return nil
}

func listenWrapperPubSubChannels(redisServerAddr string) {
	channels := []string{"sitedetail_live_updates", "wrapper_live_updates"}

	// Infinite loop
	// Logic mostly from https://godoc.org/github.com/gomodule/redigo/redis#PubSubConn
	for {

		c, err := redis.Dial(
			"tcp",
			redisServerAddr,
			redis.DialReadTimeout(readCommandTimeout),
			redis.DialWriteTimeout(writeCommandTimeout),
			redis.DialConnectTimeout(connectTimeout),
		)
		if err != nil {
			log.Printf("REDIS_PUBSUB: error connecting to %s: %v", redisServerAddr, err)
			log.Printf("REDIS_PUBSUB: will attempt to retry in %s...", retryTimeout)
			time.Sleep(retryTimeout)
			continue
		}
		defer c.Close()

		psc := redis.PubSubConn{Conn: c}

		if err := psc.Subscribe(redis.Args{}.AddFlat(channels)...); err != nil {
			log.Printf("REDIS_PUBSUB: error subscribing to %v: %v", channels, err)
			log.Printf("REDIS_PUBSUB: will attempt to retry in %s...", retryTimeout)
			time.Sleep(retryTimeout)
			continue
		}

		done := make(chan error, 1)

		go func() {
			for {
				switch n := psc.Receive().(type) {
				case error:
					done <- n
					return
				case redis.Message:
					if err := onReceivePubSubMessage(n.Channel, n.Data); err != nil {
						done <- err
						return
					}
				case redis.Subscription:
					switch n.Count {
					case len(channels):
						log.Printf("REDIS_PUBSUB: listening on these channels: %v", channels)
					case 0:
						done <- nil
						return
					}
				}
			}
		}()

		ticker := time.NewTicker(healthCheckPeriod)
		defer ticker.Stop()

		breakFromErrLoop := false
		for err == nil {
			select {
			case <-ticker.C:
				if err = psc.Ping(""); err != nil {
					breakFromErrLoop = true
				}
			case err = <-done:
				breakFromErrLoop = true
			}
			if breakFromErrLoop {
				break
			}
		}

		psc.Unsubscribe()

		if err != nil {
			log.Printf("REDIS_PUBSUB: error consuming from channels (%s)", channels)
			log.Println(err.Error())
			log.Printf("REDIS_PUBSUB: will attempt to retry in %s...", retryTimeout)
			time.Sleep(retryTimeout)
		}
	}
}
