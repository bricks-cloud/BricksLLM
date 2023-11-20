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
	GetUpdatedProviderSettings(updatedAt int64) ([]*provider.Setting, error)
}

type ProviderSettingsMemDb struct {
	external    ProviderSettingsStorage
	lastUpdated int64
	settings    map[string]*provider.Setting
	lock        sync.RWMutex
	done        chan bool
	interval    time.Duration
	log         *zap.Logger
}

func NewProviderSettingsMemDb(ex ProviderSettingsStorage, log *zap.Logger, interval time.Duration) (*ProviderSettingsMemDb, error) {
	m := map[string]*provider.Setting{}
	settings, err := ex.GetAllProviderSettings()
	if err != nil {
		return nil, err
	}

	numberOfSettings := 0
	var latetest int64 = -1
	for _, s := range settings {
		m[s.Id] = s
		numberOfSettings++
		if s.UpdatedAt > latetest {
			latetest = s.UpdatedAt
		}
	}

	if numberOfSettings != 0 {
		log.Sugar().Infof("provider settings memdb updated at %d with %d provider settings", latetest, numberOfSettings)
	}

	return &ProviderSettingsMemDb{
		external:    ex,
		settings:    m,
		log:         log,
		lastUpdated: latetest,
		interval:    interval,
		done:        make(chan bool),
	}, nil
}

func (mdb *ProviderSettingsMemDb) GetSetting(k string) *provider.Setting {
	v, ok := mdb.settings[k]
	if ok {
		return v
	}

	return nil
}

func (mdb *ProviderSettingsMemDb) SetSetting(s *provider.Setting) {
	mdb.lock.RLock()
	defer mdb.lock.RUnlock()

	mdb.settings[s.Id] = s
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
			lastUpdated := mdb.lastUpdated
			select {
			case <-mdb.done:
				mdb.log.Info("provider settings memdb stopped")
				return
			case <-ticker.C:
				settings, err := mdb.external.GetUpdatedProviderSettings(lastUpdated)
				if err != nil {
					stats.Incr("bricksllm.memdb.provider_settings_memdb.listen.get_updated_provider_settings_err", nil, 1)

					mdb.log.Sugar().Debugf("priovider settings memdb failed to update a provider setting: %v", err)
					continue
				}

				if len(settings) == 0 {
					continue
				}

				any := false
				numberOfUpdated := 0
				for _, setting := range settings {
					if setting.UpdatedAt > lastUpdated {
						lastUpdated = setting.UpdatedAt
					}

					existing := mdb.GetSetting(setting.Id)
					if existing == nil || setting.UpdatedAt > existing.UpdatedAt {
						mdb.log.Sugar().Infof("provider settings memdb updated a setting: %s", setting.Id)
						numberOfUpdated += 1
						any = true
						mdb.SetSetting(setting)
					}
				}

				if any {
					mdb.log.Sugar().Infof("provider settings memdb updated at %d with %d provider settings", lastUpdated, numberOfUpdated)
				}

			}
		}
	}()
}

func (mdb *ProviderSettingsMemDb) Stop() {
	mdb.log.Info("shutting down provider settings memdb...")

	mdb.done <- true
}
