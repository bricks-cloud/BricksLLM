package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	auth "github.com/bricks-cloud/bricksllm/internal/authenticator"
	"github.com/bricks-cloud/bricksllm/internal/cache"
	"github.com/bricks-cloud/bricksllm/internal/config"
	"github.com/bricks-cloud/bricksllm/internal/logger/zap"
	"github.com/bricks-cloud/bricksllm/internal/manager"
	"github.com/bricks-cloud/bricksllm/internal/message"
	"github.com/bricks-cloud/bricksllm/internal/pii"
	"github.com/bricks-cloud/bricksllm/internal/pii/amazon"
	custompolicy "github.com/bricks-cloud/bricksllm/internal/policy/custom"
	"github.com/bricks-cloud/bricksllm/internal/provider/anthropic"
	"github.com/bricks-cloud/bricksllm/internal/provider/azure"
	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
	"github.com/bricks-cloud/bricksllm/internal/provider/deepinfra"
	"github.com/bricks-cloud/bricksllm/internal/provider/openai"
	"github.com/bricks-cloud/bricksllm/internal/provider/vllm"
	"github.com/bricks-cloud/bricksllm/internal/recorder"
	"github.com/bricks-cloud/bricksllm/internal/server/web/admin"
	"github.com/bricks-cloud/bricksllm/internal/server/web/proxy"
	"github.com/bricks-cloud/bricksllm/internal/storage/memdb"
	"github.com/bricks-cloud/bricksllm/internal/storage/postgresql"
	redisStorage "github.com/bricks-cloud/bricksllm/internal/storage/redis"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/validator"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	modePtr := flag.String("m", "dev", "select the mode that bricksllm runs in")
	privacyPtr := flag.String("p", "strict", "select the privacy mode that bricksllm runs in")

	flag.Parse()

	log := zap.NewZapLogger(*modePtr)

	gin.SetMode(gin.ReleaseMode)

	cfg, err := config.LoadConfig(log)
	if err != nil {
		log.Sugar().Fatalf("cannot parse environment variables: %v", err)
	}

	err = telemetry.Init(cfg)
	if err != nil {
		log.Sugar().Fatalf("cannot connect to telemetry provider: %v", err)
	}

	// Set up OpenTelemetry.
	otelShutdown, err := telemetry.SetupOTelSDK(context.Background(), cfg)
	if err != nil {
		log.Sugar().Fatalf("cannot setup open telemetry sdk: %v", err)
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

	err = store.CreateRoutesTable()
	if err != nil {
		log.Sugar().Fatalf("error creating routes table: %v", err)
	}

	err = store.AlterRoutesTable()
	if err != nil {
		log.Sugar().Fatalf("error altering routes table: %v", err)
	}

	err = store.CreateKeysTable()
	if err != nil {
		log.Sugar().Fatalf("error creating keys table: %v", err)
	}

	err = store.AlterKeysTable()
	if err != nil {
		log.Sugar().Fatalf("error altering keys table: %v", err)
	}

	err = store.CreateCreateAtIndexForKeys()
	if err != nil {
		log.Sugar().Fatalf("error create create at index for keys: %v", err)
	}

	err = store.CreateKeyIndexForKeys()
	if err != nil {
		log.Sugar().Fatalf("error create create at index for keys: %v", err)
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

	err = store.CreatePolicyTable()
	if err != nil {
		log.Sugar().Fatalf("error creating policies table: %v", err)
	}

	err = store.CreateEventsByDayTable()
	if err != nil {
		log.Sugar().Fatalf("error creating event aggregated by day table: %v", err)
	}

	err = store.CreateUniqueIndexForEventsTable()
	if err != nil {
		log.Sugar().Fatalf("error creating unique index for event aggregated by day table: %v", err)
	}

	err = store.CreateTimeStampIndexForEventsTable()
	if err != nil {
		log.Sugar().Fatalf("error creating time stamp index for event aggregated by day table: %v", err)
	}

	err = store.CreateKeyIdIndexForEventsTable()
	if err != nil {
		log.Sugar().Fatalf("error creating key id index for event aggregated by day table: %v", err)
	}

	err = store.CreateUsersTable()
	if err != nil {
		log.Sugar().Fatalf("error creating users table: %v", err)
	}

	err = store.CreateCreatedAtIndexForUsers()
	if err != nil {
		log.Sugar().Fatalf("error creating created at index for users table: %v", err)
	}

	err = store.CreateUserIdIndexForUsers()
	if err != nil {
		log.Sugar().Fatalf("error creating user id for users table: %v", err)
	}

	cpMemStore, err := memdb.NewCustomProvidersMemDb(store, log, cfg.InMemoryDbUpdateInterval)
	if err != nil {
		log.Sugar().Fatalf("cannot initialize custom providers memdb: %v", err)
	}
	cpMemStore.Listen()

	rMemStore, err := memdb.NewRoutesMemDb(store, store, log, cfg.InMemoryDbUpdateInterval)
	if err != nil {
		log.Sugar().Fatalf("cannot initialize routes memdb: %v", err)
	}
	rMemStore.Listen()

	defaultRedisOption := func(cfg *config.Config, dbIndex int) *redis.Options {
		return &redis.Options{
			Addr:     fmt.Sprintf("%s:%s", cfg.RedisHosts, cfg.RedisPort),
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDBStartIndex + dbIndex,
			TLSConfig: &tls.Config{
				InsecureSkipVerify: cfg.RedisInsecureSkipVerify,
			},
		}
	}

	rateLimitRedisCache := redis.NewClient(defaultRedisOption(cfg, 0))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rateLimitRedisCache.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to rate limit redis cache: %v", err)
	}

	costLimitRedisCache := redis.NewClient(defaultRedisOption(cfg, 1))

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := costLimitRedisCache.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to cost limit redis cache: %v", err)
	}

	costRedisStorage := redis.NewClient(defaultRedisOption(cfg, 2))

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := costRedisStorage.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to cost limit redis storage: %v", err)
	}

	apiRedisCache := redis.NewClient(defaultRedisOption(cfg, 3))

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := apiRedisCache.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to api redis cache: %v", err)
	}

	accessRedisCache := redis.NewClient(defaultRedisOption(cfg, 4))

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := accessRedisCache.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to api redis cache: %v", err)
	}

	userRateLimitRedisCache := redis.NewClient(defaultRedisOption(cfg, 5))

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := userRateLimitRedisCache.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to user rate limit redis cache: %v", err)
	}

	userCostLimitRedisCache := redis.NewClient(defaultRedisOption(cfg, 6))

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := userCostLimitRedisCache.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to user cost limit redis cache: %v", err)
	}

	userCostRedisStorage := redis.NewClient(defaultRedisOption(cfg, 7))

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := userCostRedisStorage.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to user cost redis cache: %v", err)
	}

	userAccessRedisCache := redis.NewClient(defaultRedisOption(cfg, 8))

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := userAccessRedisCache.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to user access redis storage: %v", err)
	}

	providerSettingsRedisCache := redis.NewClient(defaultRedisOption(cfg, 9))

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := providerSettingsRedisCache.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to provider settings redis storage: %v", err)
	}

	keysRedisCache := redis.NewClient(defaultRedisOption(cfg, 10))

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := keysRedisCache.Ping(ctx).Err(); err != nil {
		log.Sugar().Fatalf("error connecting to keys redis storage: %v", err)
	}

	rateLimitCache := redisStorage.NewCache(rateLimitRedisCache, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)
	costLimitCache := redisStorage.NewCache(costLimitRedisCache, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)
	costStorage := redisStorage.NewStore(costRedisStorage, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)
	apiCache := redisStorage.NewCache(apiRedisCache, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)
	accessCache := redisStorage.NewAccessCache(accessRedisCache, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)

	userRateLimitCache := redisStorage.NewCache(userRateLimitRedisCache, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)
	userCostLimitCache := redisStorage.NewCache(userCostLimitRedisCache, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)
	userCostStorage := redisStorage.NewStore(userCostRedisStorage, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)
	userAccessCache := redisStorage.NewAccessCache(userAccessRedisCache, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)

	psCache := redisStorage.NewProviderSettingsCache(providerSettingsRedisCache, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)
	keysCache := redisStorage.NewKeysCache(keysRedisCache, cfg.RedisWriteTimeout, cfg.RedisReadTimeout)

	m := manager.NewManager(store, costLimitCache, rateLimitCache, accessCache, keysCache)
	krm := manager.NewReportingManager(costStorage, store, store)
	psm := manager.NewProviderSettingsManager(store, psCache)
	cpm := manager.NewCustomProvidersManager(store, cpMemStore)
	rm := manager.NewRouteManager(store, store, rMemStore, psm)
	pm := manager.NewPolicyManager(store, rMemStore)
	um := manager.NewUserManager(store, store)

	as, err := admin.NewAdminServer(log, *modePtr, m, krm, psm, cpm, rm, pm, um, cfg.AdminPass)
	if err != nil {
		log.Sugar().Fatalf("error creating admin http server: %v", err)
	}

	tc := openai.NewTokenCounter()
	custom.NewTokenCounter()

	as.Run()

	ce := openai.NewCostEstimator(openai.OpenAiPerThousandTokenCost, tc)

	atc, err := anthropic.NewTokenCounter()
	if err != nil {
		log.Sugar().Fatalf("error creating anthropic token counter: %v", err)
	}

	vllmtc, err := vllm.NewTokenCounter()
	if err != nil {
		log.Sugar().Fatalf("error creating vllm token counter: %v", err)
	}

	ace := anthropic.NewCostEstimator(atc)
	aoe := azure.NewCostEstimator()
	vllme := vllm.NewCostEstimator(vllmtc)
	die := deepinfra.NewCostEstimator()

	v := validator.NewValidator(costLimitCache, rateLimitCache, costStorage)
	uv := validator.NewUserValidator(userCostLimitCache, userRateLimitCache, userCostStorage)

	rec := recorder.NewRecorder(costStorage, userCostStorage, costLimitCache, userCostLimitCache, ce, store)
	rlm := manager.NewRateLimitManager(rateLimitCache, userRateLimitCache)
	a := auth.NewAuthenticator(psm, m, rm, store)

	c := cache.NewCache(apiCache)

	messageBus := message.NewMessageBus()
	eventMessageChan := make(chan message.Message)
	messageBus.Subscribe("event", eventMessageChan)

	handler := message.NewHandler(rec, log, ace, ce, vllme, aoe, v, uv, m, um, rlm, accessCache, userAccessCache)

	eventConsumer := message.NewConsumer(eventMessageChan, log, 4, handler.HandleEventWithRequestAndResponse)
	eventConsumer.StartEventMessageConsumers()

	detector, err := amazon.NewClient(cfg.AmazonRequestTimeout, cfg.AmazonConnectionTimeout, log, cfg.AmazonRegion)
	if err != nil {
		log.Sugar().Infof("error when connecting to amazon: %v", err)
	}

	scanner := pii.NewScanner(detector)
	cd := custompolicy.NewOpenAiDetector(cfg.CustomPolicyDetectionTimeout, cfg.OpenAiApiKey)

	ps, err := proxy.NewProxyServer(log, *modePtr, *privacyPtr, c, m, rm, a, psm, cpm, store, ce, ace, aoe, v, rec, messageBus, rlm, cfg.ProxyTimeout, accessCache, userAccessCache, pm, scanner, cd, die, um, cfg.RemoveUserAgent, cfg.OpenTelemetryEnabled)
	if err != nil {
		log.Sugar().Fatalf("error creating proxy http server: %v", err)
	}

	ps.Run()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	eventConsumer.Stop()
	cpMemStore.Stop()
	rMemStore.Stop()

	log.Sugar().Infof("shutting down server...")

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

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := otelShutdown(ctx); err != nil {
		log.Sugar().Debugf("otel shutdown: %v", err)
	}

	select {
	case <-ctx.Done():
		log.Info("timeout of 5 seconds")
	}

	log.Info("server exited")
}
