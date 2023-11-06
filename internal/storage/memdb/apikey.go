package memdb

import (
	"fmt"
	"sync"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/apikey"
	"go.uber.org/zap"
)

type ApiKeyStorage interface {
	GetAllApiKeys() ([]*apikey.ResponseApiKey, error)
	GetUpdatedApiKeys(interval time.Duration) ([]*apikey.ResponseApiKey, error)
}

type ApiKeyMemDb struct {
	external ApiKeyStorage
	keys     map[string]string
	lock     sync.RWMutex
	done     chan bool
	interval time.Duration
	log      *zap.Logger
}

func NewApiKeyMemDb(ex ApiKeyStorage, log *zap.Logger, interval time.Duration) (*ApiKeyMemDb, error) {
	keys := map[string]string{}

	return &ApiKeyMemDb{
		external: ex,
		keys:     keys,
		log:      log,
		interval: interval,
		done:     make(chan bool),
	}, nil
}

func (mdb *ApiKeyMemDb) GetKey(k string) string {
	v, ok := mdb.keys[k]
	if ok {
		return v
	}

	return ""
}

func (mdb *ApiKeyMemDb) SetKey(k, v string) {
	mdb.lock.RLock()
	defer mdb.lock.RUnlock()

	mdb.keys[k] = v
}

func (mdb *ApiKeyMemDb) RemoveKey(k string) {
	mdb.lock.RLock()
	defer mdb.lock.RUnlock()

	delete(mdb.keys, k)
}

func (mdb *ApiKeyMemDb) Listen() {
	ticker := time.NewTicker(mdb.interval)
	mdb.log.Info("api key memdb started listening for api key updates")

	go func() {
		for {
			select {
			case <-mdb.done:
				mdb.log.Info("api key memdb stopped")
				return
			case <-ticker.C:
				keys, err := mdb.external.GetUpdatedApiKeys(mdb.interval)
				if err != nil {
					mdb.log.Sugar().Debugf("api key memdb failed to update api keys: %v", err)
					continue
				}

				if len(keys) == 0 {
					continue
				}

				mdb.log.Sugar().Debugf("api key memdb updated at %s", time.Now().UTC().String())

				for _, k := range keys {
					if k.Provider == "openai" {
						mdb.log.Sugar().Debugf("api key memdb updated a key: %s", k.Provider)
						mdb.SetKey(fmt.Sprintf("%s-%s", k.Provider, k.KeyName), k.Key)
					}
				}
			}
		}
	}()
}

func (mdb *ApiKeyMemDb) Stop() {
	mdb.log.Info("shutting down api key memdb...")

	mdb.done <- true
}
