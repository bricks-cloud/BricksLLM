package manager

import (
	"fmt"
	"time"

	brickserrors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/util"
)

type ProviderSettingsStorage interface {
	UpdateProviderSetting(id string, setting *provider.Setting) (*provider.Setting, error)
	CreateProviderSetting(setting *provider.Setting) (*provider.Setting, error)
}

type ProviderSettingsMemStorage interface {
	GetSetting(id string) *provider.Setting
}

type ProviderSettingsManager struct {
	Storage ProviderSettingsStorage
	MemDb   ProviderSettingsMemStorage
}

func NewProviderSettingsManager(s ProviderSettingsStorage, memdb ProviderSettingsMemStorage) *ProviderSettingsManager {
	return &ProviderSettingsManager{
		Storage: s,
		MemDb:   memdb,
	}
}

func (m *ProviderSettingsManager) CreateSetting(setting *provider.Setting) (*provider.Setting, error) {
	if len(setting.Provider) == 0 {
		return nil, brickserrors.NewValidationError("provider field cannot be empty")
	}

	if setting.Provider != "openai" {
		return nil, brickserrors.NewValidationError(fmt.Sprintf("provider %s is not supported ", setting.Provider))
	}

	if len(setting.Setting) == 0 {
		return nil, brickserrors.NewValidationError("setting field cannot be empty")
	}

	setting.Id = util.NewUuid()
	setting.CreatedAt = time.Now().Unix()
	setting.UpdatedAt = time.Now().Unix()

	if setting.Provider == "openai" {
		v, ok := setting.Setting["apikey"]
		if !ok || len(v) == 0 {
			return nil, brickserrors.NewValidationError("setting for openai is not valid")
		}
	}

	return m.Storage.CreateProviderSetting(setting)
}

func (m *ProviderSettingsManager) UpdateSetting(id string, setting *provider.Setting) (*provider.Setting, error) {
	if len(id) == 0 {
		return nil, brickserrors.NewValidationError("id cannot be empty")
	}

	if len(setting.Setting) == 0 {
		return nil, brickserrors.NewValidationError("settings field cannot be empty")
	}

	setting.UpdatedAt = time.Now().Unix()

	return m.Storage.UpdateProviderSetting(id, setting)
}

func (m *ProviderSettingsManager) GetSetting(id string) (*provider.Setting, error) {
	setting := m.MemDb.GetSetting(id)

	if setting == nil {
		return nil, brickserrors.NewNotFoundError("provider setting is not found")
	}

	return setting, nil
}
