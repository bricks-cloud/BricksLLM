package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bricks-cloud/atlas/config"
	"github.com/bricks-cloud/atlas/internal/server/web"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	rawJSON := []byte(`{
		"level": "debug",
		"encoding": "json",
		"outputPaths": ["stdout", "/tmp/logs"],
		"errorOutputPaths": ["stderr"],
		"encoderConfig": {
		  "messageKey": "message",
		  "levelKey": "level",
		  "levelEncoder": "lowercase"
		}
	  }`)

	var cfg zap.Config

	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}

	logger := zap.Must(cfg.Build())
	defer logger.Sync()

	filePath := "atlas.yaml"

	c, err := config.NewConfig(filePath)
	if err != nil {
		logger.Sugar().Fatalf("error parsing yaml config %s : %w", filePath, err)
	}

	logger.Sugar().Infof("successfuly parsed atlas yaml config file from path: %s", filePath)

	ws, err := web.NewWebServer(c, logger.Sugar())
	if err != nil {
		logger.Sugar().Fatalf("error creating http server: %w", err)
	}

	ws.Run()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Sugar().Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ws.Shutdown(ctx); err != nil {
		logger.Sugar().Fatalf("server shutdown: %w", err)
	}

	select {
	case <-ctx.Done():
		logger.Sugar().Info("timeout of 5 seconds")
	}

	logger.Sugar().Info("server exited")
}
