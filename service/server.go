package service

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/Semantics3/go-crawler/service/controller"
	"github.com/Semantics3/go-crawler/types"
	"github.com/Semantics3/go-crawler/utils"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// Start web service (REST) for crawling in recrawl/discovery_crawl/realtime_api modes
func StartWebService(appC *types.Config) {
	// TestFunc(appC)
	router := echo.New()
	router.HideBanner = true
	router.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339} | ${remote_ip} | ${method} | ${uri} | ${status} | ${latency_human}\n",
	}))

	// Crawler
	router.POST("/crawl/url", controller.GetCrawlWorkflowHandler(appC))
	router.POST("/crawl/url/simple", controller.GetCrawlSimpleHandler(appC))
	router.POST("/crawl/url/screenshot", controller.GetScreenshotHandler(appC))
	router.POST("/crawl/upload/content", controller.UploadContentToS3(appC))
	router.POST("/domain/info", controller.GetDomainInfo(appC))
	router.GET("/admin/memstats", func(c echo.Context) (err error) {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"heap_total":   utils.HumanReadable(memStats.TotalAlloc),
			"heap_in_use":  utils.HumanReadable(memStats.HeapInuse),
			"heap_alloc":   utils.HumanReadable(memStats.HeapAlloc),
			"heap_system":  utils.HumanReadable(memStats.HeapSys),
			"total_system": utils.HumanReadable(memStats.Sys),
			"stack_in_use": utils.HumanReadable(memStats.StackInuse),
			"stack_system": utils.HumanReadable(memStats.StackSys),
		})
	})

	router.GET("/health", func(c echo.Context) (err error) {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	// router.Logger.Fatal(router.Start(":4310"))

	// Start server
	go func() {
		if err := router.Start(":4310"); err != nil {
			router.Logger.Info("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 10 seconds.
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := router.Shutdown(ctx); err != nil {
		router.Logger.Fatal(err)
	}

}
