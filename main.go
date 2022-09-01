package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	_ "net/http/pprof"

	"github.com/Semantics3/go-crawler/dbs"
	"github.com/Semantics3/go-crawler/service"
	servicehelper "github.com/Semantics3/go-crawler/service/helper"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"
	jobutils "github.com/Semantics3/sem3-go-crawl-utils/jobs"
	ctypes "github.com/Semantics3/sem3-go-crawl-utils/types"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
)

// VERSION specifies the release version during deployments
const VERSION = "3.0.0"

func main() {

	log.Println("Starting go crawler: ", VERSION)

	// Parse CLI args
	cliArgs := dbs.ParseCliArgs()

	if strings.Contains(cliArgs.Env, "development") {
		log.SetFlags(0)
	}

	// Initialize http client pool
	jobutils.InitializeHTTPClientPool(30)

	// Load configurations
	var err error
	appC, err := dbs.LoadConfig(cliArgs)
	if err != nil {
		log.Printf("Loading configuration failed with error: %s", err)
		os.Exit(1)
	}

	// Start profiler
	if cliArgs.Pprof == true {
		log.Println("Starting pprof on 0.0.0.0:6060")
		go func() {
			log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
		}()
	}

	// If web service requested
	if cliArgs.IsRestMode {
		log.Printf("CRAWLJOB: Starting rest mode\n")
		go service.StartWebService(appC)
	}
	// If job service mode requested
	if cliArgs.IsJobMode {
		go func() {
			log.Printf("CRAWLJOB: Starting job mode (%s)\n", cliArgs.JobType)
			jobutils.ListenForBatches(appC.WorkerID, cliArgs.JobType, appC.ConfigData.JobServer, 1, func(jobInput *ctypes.Batch) (tasksResults ctypes.TasksResults, err error) {
				tasksResults, _, err = servicehelper.CrawlJobBatchExecute(jobInput, appC, "")
				return tasksResults, err
			})
		}()
	}
	// If worker has to be run as rabbitmq consumer mode
	if cliArgs.IsConsumeMode {
		go func() {
			err = dbs.CreateConsumerConnections(cliArgs, *appC.ConfigData, appC)
			if err != nil {
				log.Printf("CRAWL_CONSUMER_ERR: Quitting on err: %v\n", err)
				os.Exit(1)
			}
		}()
	}

	// If test or test-file mode requested
	if cliArgs.IsTestMode || cliArgs.IsTestFileMode {
		urls, err := GetTestUrls(cliArgs)
		if err != nil {
			log.Printf("CRAWLCLI_QUIT: Quitting on err\n")
			os.Exit(1)
		}
		jobInput := servicehelper.JobBatchFromUrls(urls, cliArgs.JobType, "", nil)
		_, crawlResults, err := servicehelper.CrawlJobBatchExecute(jobInput, appC, "")
		for u, w := range crawlResults {
			if w.DomainInfo != nil {
				utils.PrintDomainInfo(*w.DomainInfo)
			} else {
				log.Printf("PRINTRESP_ERR: (%s) no domaininfo present\n", u)
			}
			break
		}
		utils.PrintResults(crawlResults)
	}

	if !cliArgs.IsTestMode && !cliArgs.IsTestFileMode {
		sigInt := make(chan os.Signal, 1)
		signal.Notify(sigInt, os.Interrupt)
		for range sigInt {
			log.Println("CRAWLSERVICE_SIGINT: Received SigInt.. closing soon")
			time.Sleep(5 * time.Second)
			break
		}
	}

}

// Get test urls from cli arg or filename mentioned in cliarg
func GetTestUrls(cliArgs *types.CliArgs) (urls []string, err error) {
	if cliArgs.IsTestMode {
		if cliArgs.Url == "" {
			err = cutils.PrintErr("CLITESTURL_ERR", fmt.Sprintf("no url sent"), err)
			return nil, err
		}
		urls = []string{cliArgs.Url}
	} else if cliArgs.IsTestFileMode {
		c, err := ioutil.ReadFile(cliArgs.Filename)
		if err != nil {
			err = cutils.PrintErr("CLITESTFILE_ERR", fmt.Sprintf("failed to read file %s", cliArgs.Filename), err)
			return nil, err
		}
		urls = make([]string, 0)
		us := strings.Split(string(c), "\n")
		r, _ := regexp.Compile(`^\s*$`)
		for _, u := range us {
			if u != "" && !r.MatchString(u) {
				urls = append(urls, u)
			}
		}
	}
	return urls, nil
}
