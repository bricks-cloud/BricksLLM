package memdb

import (
	"sync"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"go.uber.org/zap"
)

type Storage interface {
	GetAllKeys() ([]*key.ResponseKey, error)
	GetUpdatedKeys(interval time.Duration) ([]*key.ResponseKey, error)
}

type MemDb struct {
	external       Storage
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

	for _, k := range keys {
		hashToKeys[k.Key] = k
	}

	return &MemDb{
		external:   ex,
		hashToKeys: hashToKeys,
		log:        log,
		interval:   interval,
		done:       make(chan bool),
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
		for {
			select {
			case <-mdb.done:
				mdb.log.Info("memdb stopped")
				return
			case <-ticker.C:
				keys, err := mdb.external.GetUpdatedKeys(mdb.interval)
				if err != nil {
					mdb.log.Sugar().Debugf("memdb failed to update keys: %v", err)
					continue
				}

				if len(keys) == 0 {
					continue
				}

				mdb.log.Sugar().Debugf("memdb updated at %s", time.Now().UTC().String())

				for _, k := range keys {
					mdb.log.Sugar().Debugf("memdb updated a key: %s", k.KeyId)

					mdb.SetKey(k)
				}
			}
		}
	}()
}

func (mdb *MemDb) Stop() {
	mdb.log.Info("shutting down memdb...")

	mdb.done <- true
}
