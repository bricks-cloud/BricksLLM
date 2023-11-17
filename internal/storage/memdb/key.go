package memdb

import (
	"sync"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"go.uber.org/zap"
)

type Storage interface {
	GetAllKeys() ([]*key.ResponseKey, error)
	GetUpdatedKeys(updatedAt int64) ([]*key.ResponseKey, error)
}

type MemDb struct {
	external       Storage
	lastUpdated    int64
	hashToKeys     map[string]*key.ResponseKey
	hashToKeysLock sync.RWMutex
	done           chan bool
	interval       time.Duration
	log            *zap.Logger
}

func NewMemDb(ex Storage, log *zap.Logger, interval time.Duration) (*MemDb, error) {
	hashToKeys := map[string]*key.ResponseKey{}

	keys, err := ex.GetAllKeys()
	if err != nil {
		return nil, err
	}

	var latetest int64 = -1
	for _, k := range keys {
		hashToKeys[k.Key] = k
		if k.UpdatedAt > latetest {
			latetest = k.UpdatedAt
		}
	}

	return &MemDb{
		external:    ex,
		hashToKeys:  hashToKeys,
		log:         log,
		lastUpdated: latetest,
		interval:    interval,
		done:        make(chan bool),
	}, nil
}

func (mdb *MemDb) GetKey(hash string) *key.ResponseKey {
	k, ok := mdb.hashToKeys[hash]
	if ok {
		return k
	}

	return nil
}

func (mdb *MemDb) SetKey(k *key.ResponseKey) {
	mdb.hashToKeysLock.RLock()
	defer mdb.hashToKeysLock.RUnlock()

	mdb.hashToKeys[k.Key] = k
}

func (mdb *MemDb) RemoveKey(k *key.ResponseKey) {
	mdb.hashToKeysLock.RLock()
	defer mdb.hashToKeysLock.RUnlock()

	delete(mdb.hashToKeys, k.Key)
}

func (mdb *MemDb) Listen() {
	ticker := time.NewTicker(mdb.interval)
	mdb.log.Info("memdb started listening for key updates")

	go func() {
		lastUpdated := mdb.lastUpdated
		for {
			select {
			case <-mdb.done:
				mdb.log.Info("memdb stopped")
				return
			case <-ticker.C:
				keys, err := mdb.external.GetUpdatedKeys(lastUpdated)
				if err != nil {
					stats.Incr("bricksllm.memdb.memdb.listen.get_updated_keys_error", nil, 1)

					mdb.log.Sugar().Debugf("memdb failed to update keys: %v", err)
					continue
				}

				if len(keys) == 0 {
					continue
				}

				numberOfUpdated := 0
				for _, k := range keys {
					if k.UpdatedAt > lastUpdated {
						lastUpdated = k.UpdatedAt
					}

					existing := mdb.GetKey(k.Key)
					if k.UpdatedAt > existing.UpdatedAt {
						mdb.log.Sugar().Infof("key settings memdb updated a key: %s", k.KeyId)
						numberOfUpdated += 1
						mdb.SetKey(k)
					}
				}

				mdb.log.Sugar().Infof("key settings memdb updated at %d with %d keys settings", lastUpdated, numberOfUpdated)
			}
		}
	}()
}

func (mdb *MemDb) Stop() {
	mdb.log.Info("shutting down memdb...")

	mdb.done <- true
}
