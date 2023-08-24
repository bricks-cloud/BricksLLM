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
	"github.com/bricks-cloud/bricksllm/internal/provider/openai"
	"github.com/bricks-cloud/bricksllm/internal/recorder"
	"github.com/bricks-cloud/bricksllm/internal/server/web"
	"github.com/bricks-cloud/bricksllm/internal/storage/memdb"
	"github.com/bricks-cloud/bricksllm/internal/storage/postgresql"
	redisStorage "github.com/bricks-cloud/bricksllm/internal/storage/redis"
	"github.com/bricks-cloud/bricksllm/internal/validator"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	openAiCostPrefix      string = "openai-cost"
	openAiTotalCostPrefix string = "openai-total-cost"
	rateLimitPrefix       string = "rate-limit"
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

	e := encrypter.NewEncrypter()
	m := manager.NewManager(store, e)
	as, err := web.NewAdminServer(lg, m)
	if err != nil {
		lg.Fatalf("error creating admin http server: %v", err)
	}

	tc, err := openai.NewTokenCounter()
	if err != nil {
		lg.Fatalf("error creating token counter: %v", err)
	}

	as.Run()

	c := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHosts, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0,
	})

	rc := redisStorage.NewCache(c, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)
	rs := redisStorage.NewStore(c, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)
	ce := openai.NewCostEstimator(openai.OpenAiPerThousandTokenCost, tc)
	v := validator.NewValidator(rc, rs, openAiCostPrefix, openAiTotalCostPrefix, rateLimitPrefix)
	rec := recorder.NewRecorder(rs, ce, openAiTotalCostPrefix)

	ps, err := web.NewProxyServer(lg, m, store, memStore, ce, v, rec, cfg.OpenAiKey, e)
	if err != nil {
		lg.Fatalf("error creating proxy http server: %v", err)
	}

	ps.Run()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	memStore.Stop()

	lg.Infof("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := as.Shutdown(ctx); err != nil {
		lg.Debugf("admin server shutdown: %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ps.Shutdown(ctx); err != nil {
		lg.Debugf("proxy server shutdown: %v", err)
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
