package discovery

import (
	"encoding/xml"
	"fmt"
	"log"

	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	jobutils "github.com/Semantics3/sem3-go-crawl-utils/jobs"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
)

// Ref: https://github.com/yterajima/go-sitemap

// Index is a structure of <sitemapindex>
type Index struct {
	XMLName xml.Name `xml:"sitemapindex"`
	Sitemap []parts  `xml:"sitemap"`
}

// parts is a structure of <sitemap> in <sitemapindex>
type parts struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

// Sitemap is a structure of <sitemap>
type Sitemap struct {
	XMLName xml.Name `xml:"urlset"`
	URL     []URL    `xml:"url"`
}

// URL is a structure of <url> in <sitemap>
type URL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod"`
	ChangeFreq string  `xml:"changefreq"`
	Priority   float32 `xml:"priority"`
}

// Formats size of a digital entity
// Ref: https://programming.guide/go/formatting-byte-size-to-human-readable-format.html
func formatDigitalSize(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

func ExtractSitemapUrls(url string, webResponseContent string) ([]string, error) {
	var smap Sitemap
	var idx Index

	log.Printf("SITEMAP_PARSER: Size of sitemap: %s\n", formatDigitalSize(int64(len(webResponseContent))))

	smapErr := xml.Unmarshal([]byte(webResponseContent), &smap)
	idxErr := xml.Unmarshal([]byte(webResponseContent), &idx)

	if smapErr != nil && idxErr != nil {
		return []string{}, fmt.Errorf("SITEMAP_PARSE_FAILED: %s is neither a sitemap nor a sitemap index", url)
	}

	if smapErr == nil {
		return extractLocationsFromURLs(smap.URL), nil
	} else {
		return extractLocationsFromIndexParts(idx.Sitemap), nil
	}
}

func extractLocationsFromIndexParts(parts []parts) []string {
	var locations []string
	for _, part := range parts {
		locations = append(locations, part.Loc)
	}
	return locations
}

func extractLocationsFromURLs(urls []URL) []string {
	var locations []string
	for _, url := range urls {
		locations = append(locations, url.Loc)
	}
	return locations
}

func LoadTasksToJobServer(jobId string, feedbackLinks map[string]ctypes.UrlMetadata, workflow *types.CrawlWorkflow, appC *types.Config) {
	tasks := make([]string, 0)
	for task, _ := range feedbackLinks {
		tasks = append(tasks, task)
	}
	utils.BatchProcessItems(tasks, 25, func(batch []string) (err error) {
		_, err = jobutils.LoadData(batch, jobId, "discovery-bot", appC.ConfigData.JobServer)
		if err != nil {
			log.Println(err)
		}
		return nil
	})
	workflow.Data.Links = make(map[string]ctypes.UrlMetadata)
	return
}
