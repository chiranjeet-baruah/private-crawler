package sources

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Semantics3/go-crawler/utils"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
	"github.com/Semantics3/sem3-go-data-consumer/rpc"
	"github.com/gomodule/redigo/redis"
)

type hash = map[string]interface{}

// MakeRPCRequest - Makes an RPC request and handles the response
func MakeRPCRequest(rpcClient *rpc.RPCClient, mode string, url string, method string, args []interface{}, dest interface{}) error {

	start := time.Now()
	resp, err := rpcClient.Call(method, args...)
	if err != nil {
		return cutils.PrintErr("EXTRACTION_FAILED_RPC", fmt.Sprintf("failed %s rpc call for %s", mode, url), err)
	}

	logMessage := fmt.Sprintf("EXTRACTION_RPC_RESPONSE: URL: %s, MODE: %s, METHOD: %s, RoundTrip: %f", url, mode, method, utils.ComputeDuration(start))
	utils.PrintResponseDetails(200, logMessage)

	r := resp.(map[string]interface{})

	if r["status"] != nil {
		status := r["status"].(float64)
		if status == 0 {
			failureMessage := r["message"].(string)
			return cutils.PrintErr("EXTRACTION_FAILED_CE", fmt.Sprintf("%s rpc failed for %s", mode, url), failureMessage)
		}
	} else if r["error"] != nil {
		failureMessage := r["error"].(string)
		return cutils.PrintErr("EXTRACTION_FAILED_CE", fmt.Sprintf("%s rpc failed for %s", mode, url), failureMessage)
	}

	if r["result"] != nil {
		r = r["result"].(map[string]interface{})
	}

	err = parseExtractionRPCResponse(url, r, dest)
	return err
}

// Unmarshal a generic RPC response to a specific type
func parseExtractionRPCResponse(url string, src interface{}, dst interface{}) error {
	jsonBytes, err := json.Marshal(src)

	if err != nil {
		err = cutils.PrintErr("EXTRACTION_JSON_MARSHAL_FAILED", fmt.Sprintf("failed to marshal extraction response (URL: %s, SOURCE: %v)", url, src), err)
		return err
	}

	err = json.Unmarshal(jsonBytes, dst)
	if err != nil {
		err = cutils.PrintErr("EXTRACTION_JSON_UNMARSHAL_FAILED", fmt.Sprintf("failed to unmarshal extraction response (URL: %s, SOURCE: %s)", url, string(jsonBytes)), err)
		return err
	}

	return nil
}

// CheckRateLimitPerSecond checks if the rate limit on source is not exceeded
func CheckRateLimitPerSecond(client *redis.Pool, source string) (bool, error) {
	// set key
	if source == "" {
		return false, fmt.Errorf("source not provided")
	}
	conn := client.Get()
	defer conn.Close()

	rateLimitKey := fmt.Sprintf("global_ratelimit_per_second_%s", source) // amazon and m101
	limit, err := redis.Int64(conn.Do("GET", rateLimitKey))
	if err != nil {
		return false, err
	}

	currTime := time.Now().Unix()
	key := fmt.Sprintf("ratelimit_per_second_%s_%d", source, currTime)
	// get number of requests already made
	conn.Send("MULTI")
	conn.Send("INCR", key)
	conn.Send("EXPIRE", key, 10)
	res, err := redis.Values(conn.Do("EXEC"))

	if err != nil {
		return false, err
	}
	if requestsMade, ok := res[0].(int64); ok {
		// log.Println("AMAZON_RATELIMIT_VARIABLES", requestsMade, limit, currTime)
		if requestsMade > limit {
			return false, nil
		}
	}
	return true, nil
}
