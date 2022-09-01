package utils

import (
	"fmt"
	"log"

	"github.com/Semantics3/sem3-go-crawl-utils/sitedetails"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

// IsSitemapURL checks if a given string is a sitemap URL
func IsSitemapURL(url string) bool {
	likeSitemapURL, _ := cutils.ApplyRegex(url, url, "sitemapRegex", `(?:\.axd|\.xml|\.gz|sitemap\.cfm|sitemap\.ashx)`, "")
	return IsURL(url) && likeSitemapURL
}

// IsURL checks if the given string is a URL
func IsURL(s string) bool {
	isValidURL, _ := cutils.ApplyRegex(s, s, "validURLRegex", `^https?\:\/\/`, "")
	if isValidURL && len(s) >= 7 && len(s) <= 1000 {
		return true
	}
	return false
}

// IsProductURL checks if the given URL is a product URL
func IsProductURL(url string, site string, sitedetail *ctypes.Sitedetail) bool {
	t := false
	urlFilterStrs, err := cutils.InterfaceToStrArray(sitedetail.URLFilters)
	if err != nil {
		err = cutils.PrintErr("DOMAININFO_URLFILTERERR", fmt.Sprintf("unknown url filters for site %s", site), err)
		log.Println(err)
	}
	matchIndex, err := cutils.ApplyRegexes(url, url, "urlFilters", urlFilterStrs)
	if err != nil {
		err = cutils.PrintErr("DOMAININFO_URLFILTER_APPLYERR", fmt.Sprintf("failed to apply urlFilters for %s", site), err)
		log.Println(err)
	}
	if matchIndex > -1 {
		t = true
	}

	return t
}

func GetSkuFromURL(url string, sitedetail *ctypes.Sitedetail) (string, error) {
	curl, err := sitedetails.CanonicalizeUrl(url, sitedetail)
	if err != nil {
		err = cutils.PrintErr("CANONICALIZEERR", fmt.Sprintf("failed to canonicalize %s", url), err)
		return "", err
	}
	sku, match, err := sitedetails.ApplySitedetailSkuRegex(curl, sitedetail)
	if err != nil {
		err = cutils.PrintErr("SKUREGEXERR", fmt.Sprintf("failed to extract sku from %s", curl), err)
		return "", err
	}

	if !match {
		err = cutils.PrintErr("NO_SKU_MATCH_FOUND", fmt.Sprintf("No sku match found for %s", curl), "")
		return "", err
	}

	return sku, nil
}
