package manager

import (
	"errors"
	"fmt"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/apikey"
	"github.com/bricks-cloud/bricksllm/internal/util"
)

type ApiKeyStorage interface {
	UpsertApiKey(rk *apikey.RequestApiKey) error
}

type ApiKeyMemStorage interface {
	GetKey(k string) string
}

type ApiKeyManager struct {
	Storage ApiKeyStorage
	MemDb   ApiKeyMemStorage
}

func NewApiKeyManager(s ApiKeyStorage, memdb ApiKeyMemStorage) *ApiKeyManager {
	return &ApiKeyManager{
		Storage: s,
		MemDb:   memdb,
	}
}

func (akm *ApiKeyManager) SetKey(key *apikey.RequestApiKey) error {
	if key.Provider != "openai" {
		return fmt.Errorf("provider %s is not supported", key.Provider)
	}

	if key.KeyName != "apikey" {
		return errors.New("openai requires an apikey")
	}

	key.Id = util.NewUuid()
	key.UpdatedAt = time.Now().Unix()

	if err := akm.Storage.UpsertApiKey(key); err != nil {
		return err
	}

	return nil
}

func (akm *ApiKeyManager) GetKey(provider string, keyName string) (string, error) {
	key := akm.MemDb.GetKey(fmt.Sprintf("%s-%s", provider, keyName))

	if len(key) == 0 {
		return "", errors.New("api key not found")
	}

	return key, nil
}
