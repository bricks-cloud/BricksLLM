package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/config"
	"github.com/bricks-cloud/bricksllm/internal/encrypter"
	"github.com/bricks-cloud/bricksllm/internal/logger/zap"
	"github.com/bricks-cloud/bricksllm/internal/manager"
	"github.com/bricks-cloud/bricksllm/internal/server/web"
	"github.com/bricks-cloud/bricksllm/internal/storage/memdb"
	"github.com/bricks-cloud/bricksllm/internal/storage/postgresql"
	"github.com/gin-gonic/gin"
)

func main() {
	modePtr := flag.String("m", "dev", "select the mode that bricksllm runs in")
	lg := zap.NewLogger(*modePtr)

	gin.SetMode(gin.ReleaseMode)

	cfg, err := config.ParseEnvVariables()
	if err != nil {
		lg.Fatalf("cannot parse environment variables: %v", err)
	}

	sslModeSuffix := ""
	if !cfg.PostgresqlSslEnabled {
		sslModeSuffix = "?sslmode=disable"
	}

	store, err := postgresql.NewStore(
		fmt.Sprintf("postgresql://%s:%s@%s:%s/postgres%s", cfg.PostgresqlUsername, cfg.PostgresqlUsername, cfg.PostgresqlHosts, cfg.PostgresqlPort, sslModeSuffix),
		lg,
		cfg.PostgresqlWriteTimeout,
		cfg.PostgresqlReadTimeout,
	)

	if err != nil {
		lg.Fatalf("cannot connect to postgresql: %v", err)
	}

	err = store.CreateKeysTable()
	if err != nil {
		lg.Fatalf("error creating keys table: %v", err)
	}

	memStore, err := memdb.NewMemDb(store, lg, cfg.InMemoryDbUpdateInterval)
	if err != nil {
		lg.Fatalf("cannot initialize memdb: %v", err)
	}

	memStore.Listen()

	e := encrypter.NewEncrypter(cfg.EncryptionKey)
	m := manager.NewManager(store, e)
	as, err := web.NewAdminServer(lg, m)
	if err != nil {
		lg.Fatalf("error creating http server: %v", err)
	}

	as.Run()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	memStore.Stop()
	lg.Infof("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := as.Shutdown(ctx); err != nil {
		lg.Fatalf("server shutdown: %v", err)
	}

	select {
	case <-ctx.Done():
		lg.Infof("timeout of 5 seconds")
	}

	err = store.DropKeysTable()
	if err != nil {
		lg.Fatalf("error dropping keys table: %v", err)
	}

	lg.Infof("server exited")
}
