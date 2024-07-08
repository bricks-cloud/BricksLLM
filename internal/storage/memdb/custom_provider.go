package memdb

import (
	"sync"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"go.uber.org/zap"
)

type CustomProvidersStorage interface {
	GetCustomProviders() ([]*custom.Provider, error)
	GetUpdatedCustomProviders(updatedAt int64) ([]*custom.Provider, error)
}

type CustomProvidersMemDb struct {
	external        CustomProvidersStorage
	lastUpdated     int64
	nameToProviders map[string]*custom.Provider
	lock            sync.RWMutex
	done            chan bool
	interval        time.Duration
	log             *zap.Logger
}

func NewCustomProvidersMemDb(ex CustomProvidersStorage, log *zap.Logger, interval time.Duration) (*CustomProvidersMemDb, error) {
	nameToProviders := map[string]*custom.Provider{}

	providers, err := ex.GetCustomProviders()
	if err != nil {
		return nil, err
	}

	numberOfProviders := 0
	var latetest int64 = -1
	for _, p := range providers {
		nameToProviders[p.Provider] = p
		numberOfProviders++
		if p.UpdatedAt > latetest {
			latetest = p.UpdatedAt
		}
	}

	if numberOfProviders != 0 {
		log.Sugar().Infof("custom provider settings memdb updated at %d with %d providers", latetest, numberOfProviders)
	}

	return &CustomProvidersMemDb{
		external:        ex,
		nameToProviders: nameToProviders,
		log:             log,
		lastUpdated:     latetest,
		interval:        interval,
		done:            make(chan bool),
	}, nil
}

func (mdb *CustomProvidersMemDb) GetProvider(name string) *custom.Provider {
	provider, ok := mdb.nameToProviders[name]
	if ok {
		return provider
	}

	return nil
}

func (mdb *CustomProvidersMemDb) GetRouteConfig(name string, path string) *custom.RouteConfig {
	provider, ok := mdb.nameToProviders[name]
	if ok {
		for _, rc := range provider.RouteConfigs {
			if rc.Path == path {
				return rc
			}
		}
	}

	return nil
}

func (mdb *CustomProvidersMemDb) SetProvider(provider *custom.Provider) {
	mdb.lock.RLock()
	defer mdb.lock.RUnlock()

	mdb.nameToProviders[provider.Provider] = provider
}

func (mdb *CustomProvidersMemDb) Listen() {
	ticker := time.NewTicker(mdb.interval)
	mdb.log.Info("custom providers memdb started listening for provider updates")

	go func() {
		lastUpdated := mdb.lastUpdated
		for {
			select {
			case <-mdb.done:
				mdb.log.Info("memdb stopped")
				return
			case <-ticker.C:
				providers, err := mdb.external.GetUpdatedCustomProviders(lastUpdated)
				if err != nil {
					telemetry.Incr("bricksllm.memdb.custom_proivders_memdb.listen.get_updated_custom_providers_error", nil, 1)

					mdb.log.Sugar().Debugf("memdb failed to update custom providers: %v", err)
					continue
				}

				if len(providers) == 0 {
					continue
				}

				any := false
				numberOfUpdated := 0
				for _, provider := range providers {
					if provider.UpdatedAt > lastUpdated {
						lastUpdated = provider.UpdatedAt
					}

					existing := mdb.GetProvider(provider.Provider)
					if existing == nil || provider.UpdatedAt > existing.UpdatedAt {
						mdb.log.Sugar().Infof("custom providers memdb updated a provider: %s", provider.Id)
						numberOfUpdated += 1
						any = true
						mdb.SetProvider(provider)
					}
				}

				if any {
					mdb.log.Sugar().Infof("custom providers memdb updated at %d with %d providers", lastUpdated, numberOfUpdated)
				}
			}
		}
	}()
}

func (mdb *CustomProvidersMemDb) Stop() {
	mdb.log.Info("shutting down custom providers memdb...")

	mdb.done <- true
}
