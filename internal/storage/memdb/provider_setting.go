package memdb

import (
	"sync"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"go.uber.org/zap"
)

type ProviderSettingsStorage interface {
	GetAllProviderSettings() ([]*provider.Setting, error)
	GetUpdatedProviderSettings(interval time.Duration) ([]*provider.Setting, error)
}

type ProviderSettingsMemDb struct {
	external ProviderSettingsStorage
	settings map[string]*provider.Setting
	lock     sync.RWMutex
	done     chan bool
	interval time.Duration
	log      *zap.Logger
}

func NewProviderSettingsMemDb(ex ProviderSettingsStorage, log *zap.Logger, interval time.Duration) (*ProviderSettingsMemDb, error) {
	m := map[string]*provider.Setting{}
	settings, err := ex.GetAllProviderSettings()
	if err != nil {
		return nil, err
	}

	for _, s := range settings {
		m[s.Id] = s
	}

	return &ProviderSettingsMemDb{
		external: ex,
		settings: m,
		log:      log,
		interval: interval,
		done:     make(chan bool),
	}, nil
}

func (mdb *ProviderSettingsMemDb) GetSetting(k string) *provider.Setting {
	v, ok := mdb.settings[k]
	if ok {
		return v
	}

	return nil
}

func (mdb *ProviderSettingsMemDb) SetSetting(k string, v *provider.Setting) {
	mdb.lock.RLock()
	defer mdb.lock.RUnlock()

	mdb.settings[k] = v
}

func (mdb *ProviderSettingsMemDb) RemoveSetting(k string) {
	mdb.lock.RLock()
	defer mdb.lock.RUnlock()

	delete(mdb.settings, k)
}

func (mdb *ProviderSettingsMemDb) Listen() {
	ticker := time.NewTicker(mdb.interval)
	mdb.log.Info("provider settings memdb started listening for provider setting updates")

	go func() {
		for {
			select {
			case <-mdb.done:
				mdb.log.Info("provider settings memdb stopped")
				return
			case <-ticker.C:
				settings, err := mdb.external.GetUpdatedProviderSettings(mdb.interval)
				if err != nil {
					stats.Incr("bricksllm.memdb.provider_settings_memdb.listen.get_updated_provider_settings", nil, 1)

					mdb.log.Sugar().Debugf("priovider settings memdb failed to update a provider setting: %v", err)
					continue
				}

				if len(settings) == 0 {
					continue
				}

				mdb.log.Sugar().Debugf("provider settings memdb updated at %s", time.Now().UTC().String())

				for _, setting := range settings {
					mdb.SetSetting(setting.Id, setting)
				}
			}
		}
	}()
}

func (mdb *ProviderSettingsMemDb) Stop() {
	mdb.log.Info("shutting down provider settings memdb...")

	mdb.done <- true
}
