package utils

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

// Pretty print JSON
func PrettyJSON(key string, val interface{}, canPrint bool) (jsonStr string, err error) {
	jsonBytes, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		log.Printf("JSON_ENCODEERR: Error encoding json: (%v %v)\n", val, err)
		return "", err
	}

	jsonStr = string(jsonBytes)
	if canPrint {
		log.Printf("KEY: %s, VALUE: %v\n", key, jsonStr)
	}
	return jsonStr, nil
}

func ComputeDuration(start time.Time) float64 {
	end := time.Now()
	duration := end.Sub(start)
	return duration.Seconds()
}

func ParseJobParams(url string, cjp map[string]interface{}) (*ctypes.CrawlJobParams, error) {

	var jobParams ctypes.CrawlJobParams

	jsonBytes, err := json.Marshal(cjp)
	if err != nil {
		err = cutils.PrintErr("UTILS_JOBPARAMSERR", fmt.Sprintf("URL: (%s), Error marshalling input job params: %v", url, cjp), err)
		return nil, err
	}

	err = json.Unmarshal(jsonBytes, &jobParams)
	if err != nil {
		err = cutils.PrintErr("UTILS_JOBPARAMSERR", fmt.Sprintf("URL: (%s), Error unmarshalling job params: %v", url, cjp), err)
		return nil, err
	}

	return &jobParams, nil
}

// Print err on return
func ReturnErr(prefix string, item string, err error) {
	if err != nil {
		log.Printf("%s: (Item %s) %v", prefix, item, err)
	}
}

func GetStrPtr(val string) (valPtr *string) {
	return &val
}

func GetIntFromStrPtr(prefix string, val *string) (valInt int) {
	if val != nil && *val != "" {
		i, err := strconv.Atoi(*val)
		if err != nil {
			log.Printf("%s (%s): %v\n", prefix, *val, err)
			return valInt
		}
		valInt = i
	}
	return valInt
}

// Get geo id from wrapper if present (default value: 0)
func GetGeoIdFromWrapper(wrapper *ctypes.Wrapper) (geoId int) {
	var geo int
	wrapperContent := wrapper.Content
	for _, content := range wrapperContent {
		if content.Name == "products" {
			for _, item := range content.Entities {
				name, _ := cutils.GetStringKey(item, "name")
				if name == "geo_id" {
					geo, _ = cutils.GetIntKey(item, "set")
				}
			}
		}
	}
	return geo
}

// Generate random string of length l
func GenerateUniqueId(l int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(65, 90))
	}
	return string(bytes)
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func BatchProcessItems(items []string, batchSize int, fn func([]string) error) {

	total := len(items)
	if total >= 1500 {
		batchSize = 50
	}

	numBatches := (total / batchSize) + 1
	for i := 0; i < numBatches; i++ {
		start := i * batchSize
		end := start + batchSize
		if end >= total {
			end = total
		}
		batch := items[start:end]
		err := fn(batch)
		if err != nil {
			PrettyJSON("BATCH_PROCESSING", batch, true)
			log.Printf("Batch processing failed with error %v\n", err)
			continue
		}
		log.Printf("BATCH_PROCESSING: Batch: %d, Items: %d/%d\n", i+1, end, total)
	}
}

// Apply regex and return all matching capture groups
func FindStringSubmatch(content string, regexStr string, flags string) (matches []string, didMatch bool, err error) {
	cr, err := cutils.GetCompiledRegex(regexStr, flags)
	if err != nil {
		return []string{}, false, err
	}
	matches = cr.FindStringSubmatch(content)
	if len(matches) > 1 {
		didMatch = true
		log.Printf("FIND_STRING_SUBMATCH: Item %s matches regex %s\n", content, regexStr)
	}
	return matches, didMatch, nil
}

// Get md5 sum of a string
func GetMD5Sum(val string) int64 {
	h := md5.New()
	var seed uint64 = binary.BigEndian.Uint64(h.Sum(nil))
	return int64(seed)
}

// Convert bytes to human readable form
func HumanReadable(num uint64) string {
	unit := "Bytes"
	val := float64(num)

	if val > 1024 {
		val = val / 1024
		unit = "KB"
	}

	if val > 1024 {
		val = val / 1024
		unit = "MB"
	}

	if val > 1024 {
		val = val / 1024
		unit = "GB"
	}

	res := fmt.Sprintf("%.2f %s", val, unit)
	return res
}
