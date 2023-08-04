package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bricks-cloud/bricksllm/config"
	"github.com/bricks-cloud/bricksllm/internal/logger/zap"
	"github.com/bricks-cloud/bricksllm/internal/server/web"
	"github.com/gin-gonic/gin"
)

func main() {
	modePtr := flag.String("m", "dev", "select the mode that bricksllm runs in")

	filePathPtr := flag.String("c", "", "enter the file path to the config file")

	flag.Parse()

	gin.SetMode(gin.ReleaseMode)

	logger := zap.NewLogger(*modePtr)

	logger.Infof("running bricksllm in %s mode", *modePtr)

	defer logger.Sync()

	if filePathPtr == nil || len(*filePathPtr) == 0 {
		logger.Fatal("path is not specified")
	}

	filePath := *filePathPtr

	c, err := config.NewConfig(filePath)
	if err != nil {
		logger.Fatalf("error parsing yaml config: %v", err)
	}

	logger.Infof("successfuly parsed bricksllm yaml config file from path: %s", filePath)

	ws, err := web.NewWebServer(c, logger, *modePtr)
	if err != nil {
		logger.Fatalf("error creating http server: %v", err)
	}

	ws.Run()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Infof("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ws.Shutdown(ctx); err != nil {
		logger.Fatalf("server shutdown: %v", err)
	}

	select {
	case <-ctx.Done():
		logger.Infof("timeout of 5 seconds")
	}

	logger.Infof("server exited")
}
