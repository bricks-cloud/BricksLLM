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
	logger "github.com/bricks-cloud/bricksllm/internal/logger/zap"
	"github.com/bricks-cloud/bricksllm/internal/manager"
	"github.com/bricks-cloud/bricksllm/internal/provider/anthropic"
	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
	"github.com/bricks-cloud/bricksllm/internal/provider/openai"
	"github.com/bricks-cloud/bricksllm/internal/recorder"
	"github.com/bricks-cloud/bricksllm/internal/server/web/admin"
	"github.com/bricks-cloud/bricksllm/internal/server/web/proxy"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/bricks-cloud/bricksllm/internal/storage/memdb"
	"github.com/bricks-cloud/bricksllm/internal/storage/postgresql"
	redisStorage "github.com/bricks-cloud/bricksllm/internal/storage/redis"
	"github.com/bricks-cloud/bricksllm/internal/validator"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	modePtr := flag.String("m", "dev", "select the mode that bricksllm runs in")
	privacyPtr := flag.String("p", "strict", "select the privacy mode that bricksllm runs in")
	flag.Parse()

	log := logger.NewZapLogger(*modePtr)

	gin.SetMode(gin.ReleaseMode)

	cfg, err := config.ParseEnvVariables()
	if err != nil {
		log.Sugar().Fatalf("cannot parse environment variables: %v", err)
	}

	err = stats.InitializeClient(cfg.StatsProvider)
	if err != nil {
		log.Sugar().Fatalf("cannot connect to telemetry provider: %v", err)
	}

	store, err := postgresql.NewStore(
		fmt.Sprintf("postgresql:///%s?sslmode=%s&user=%s&password=%s&host=%s&port=%s", cfg.PostgresqlDbName, cfg.PostgresqlSslMode, cfg.PostgresqlUsername, cfg.PostgresqlPassword, cfg.PostgresqlHosts, cfg.PostgresqlPort),
		cfg.PostgresqlWriteTimeout,
		cfg.PostgresqlReadTimeout,
	)

	if err != nil {
		log.Sugar().Fatalf("cannot connect to postgresql: %v", err)
	}

	err = store.CreateCustomProvidersTable()
	if err != nil {
		log.Sugar().Fatalf("error creating custom providers table: %v", err)
	}

	err = store.CreateKeysTable()
	if err != nil {
		log.Sugar().Fatalf("error creating keys table: %v", err)
	}

	err = store.AlterKeysTable()
	if err != nil {
		log.Sugar().Fatalf("error altering keys table: %v", err)
	}

	err = store.CreateEventsTable()
	if err != nil {
		log.Sugar().Fatalf("error creating events table: %v", err)
	}

	err = store.AlterEventsTable()
	if err != nil {
		log.Sugar().Fatalf("error altering events table: %v", err)
	}

	err = store.CreateProviderSettingsTable()
	if err != nil {
		log.Sugar().Fatalf("error creating provider settings table: %v", err)
	}

	err = store.AlterProviderSettingsTable()
	if err != nil {
		log.Sugar().Fatalf("error altering provider settings table: %v", err)
	}

	memStore, err := memdb.NewMemDb(store, log, cfg.InMemoryDbUpdateInterval)
	if err != nil {
		log.Sugar().Fatalf("cannot initialize memdb: %v", err)
	}
	memStore.Listen()

	psMemStore, err := memdb.NewProviderSettingsMemDb(store, log, cfg.InMemoryDbUpdateInterval)
	if err != nil {
		log.Sugar().Fatalf("cannot initialize provider settings memdb: %v", err)
	}
	psMemStore.Listen()

	cpMemStore, err := memdb.NewCustomProvidersMemDb(store, log, cfg.InMemoryDbUpdateInterval)
	if err != nil {
		log.Sugar().Fatalf("cannot initialize custom providers memdb: %v", err)
	}
	cpMemStore.Listen()

	rateLimitRedisCache := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHosts, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rateLimitRedisCache.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to rate limit redis cache: %v", err)
	}

	costLimitRedisCache := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHosts, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       1,
	})

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := costLimitRedisCache.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to cost limit redis cache: %v", err)
	}

	costRedisStorage := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHosts, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       2,
	})

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := costRedisStorage.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to cost limit redis storage: %v", err)
	}

	rateLimitCache := redisStorage.NewCache(rateLimitRedisCache, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)
	costLimitCache := redisStorage.NewCache(costLimitRedisCache, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)
	costStorage := redisStorage.NewStore(costRedisStorage, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)

	e := encrypter.NewEncrypter()
	m := manager.NewManager(store, e)
	krm := manager.NewReportingManager(costStorage, store, store)
	psm := manager.NewProviderSettingsManager(store, psMemStore)
	cpm := manager.NewCustomProvidersManager(store, cpMemStore)

	as, err := admin.NewAdminServer(log, *modePtr, m, krm, psm, cpm)
	if err != nil {
		log.Sugar().Fatalf("error creating admin http server: %v", err)
	}

	tc := openai.NewTokenCounter()
	custom.NewTokenCounter()
	atc, err := anthropic.NewTokenCounter()
	if err != nil {
		log.Sugar().Fatalf("error creating anthropic token counter: %v", err)
	}

	as.Run()

	ce := openai.NewCostEstimator(openai.OpenAiPerThousandTokenCost, tc)
	ae := anthropic.NewCostEstimator(atc)
	v := validator.NewValidator(costLimitCache, rateLimitCache, costStorage)
	rec := recorder.NewRecorder(costStorage, costLimitCache, ce, store)
	rlm := manager.NewRateLimitManager(rateLimitCache)

	ps, err := proxy.NewProxyServer(log, *modePtr, *privacyPtr, m, psm, cpm, store, memStore, ce, ae, v, rec, cfg.OpenAiKey, e, rlm, cfg.ProxyTimeout)
	if err != nil {
		log.Sugar().Fatalf("error creating proxy http server: %v", err)
	}

	ps.Run()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	memStore.Stop()
	psMemStore.Stop()
	cpMemStore.Stop()

	log.Sugar().Info("shutting down server...")

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := as.Shutdown(ctx); err != nil {
		log.Sugar().Debugf("admin server shutdown: %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ps.Shutdown(ctx); err != nil {
		log.Sugar().Debugf("proxy server shutdown: %v", err)
	}

	select {
	case <-ctx.Done():
		log.Sugar().Infof("timeout of 5 seconds")
	}

	err = store.DropKeysTable()
	if err != nil {
		log.Sugar().Fatalf("error dropping keys table: %v", err)
	}

	err = store.DropEventsTable()
	if err != nil {
		log.Sugar().Fatalf("error dropping events table: %v", err)
	}

	err = store.DropCustomProvidersTable()
	if err != nil {
		log.Sugar().Fatalf("error dropping custom providers table: %v", err)
	}

	err = store.DropProviderSettingsTable()
	if err != nil {
		log.Sugar().Fatalf("error dropping provider settings table: %v", err)
	}

	log.Sugar().Infof("server exited")
}
