package paapi5

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"regexp"

	"github.com/Semantics3/go-crawler/sources/amazon/paapi5/types"
)

// regexCapture get capture groups
func regexCapture(rgx string, src string) []string {
	Re := regexp.MustCompile(rgx)
	res := Re.FindStringSubmatch(src)
	return res
}

func hashedString(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// asinFromPath get product ASIN from url
func asinFromPath(path string) (asin string, err error) {
	rgx := `\/dp\/([a-zA-Z0-9]{10})`
	capturedVals := regexCapture(rgx, path)
	if len(capturedVals) < 2 {
		return "", fmt.Errorf("ASIN could not be captured from path: %s", path)
	}
	return capturedVals[1], nil
}

func writeFile(filename string, data []byte) (err error) {
	err = ioutil.WriteFile(filename, data, 0755)
	if err != nil {
		return
	}

	abs, err := filepath.Abs(fmt.Sprintf("./%s", filename))
	if err == nil {
		fmt.Println("Absolute File Path:", abs)
	}
	return
}

func readFile(filename string) (data []byte, err error) {
	data, err = ioutil.ReadFile(filename)
	return
}

func GetLocaleAsinFromURL(productURL string) (locale types.Locale, asin string, code string, err error) {
	parsedURL, err := url.Parse(productURL)
	if err != nil {
		return locale, asin, "URL_INVALID", fmt.Errorf("INVALID_URL_ERR: %v", err)
	}

	locale, err = types.NewLocale(parsedURL.Hostname())
	if err != nil {
		return locale, asin, "DOMAIN_NOT_SUPPORTED_FOR_SOURCE", err
	}

	asin, err = asinFromPath(parsedURL.Path)
	if err != nil {
		return locale, asin, "ASIN_NOT_FOUND", err
	}

	return
}
