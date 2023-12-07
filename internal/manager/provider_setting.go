package manager

import (
	"fmt"
	"time"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
	"github.com/bricks-cloud/bricksllm/internal/util"
)

type ProviderSettingsStorage interface {
	UpdateProviderSetting(id string, setting *provider.UpdateSetting) (*provider.Setting, error)
	CreateProviderSetting(setting *provider.Setting) (*provider.Setting, error)
	GetProviderSetting(id string) (*provider.Setting, error)
	GetCustomProviderByName(name string) (*custom.Provider, error)
	GetProviderSettings(withSecret bool) ([]*provider.Setting, error)
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

func (m *ProviderSettingsManager) validateSettings(name string, setting *provider.Setting) error {
	if setting.Provider != "openai" && setting.Provider != "anthropic" {
		provider, err := m.Storage.GetCustomProviderByName(name)
		_, ok := err.(notFoundError)
		if ok {
			return internal_errors.NewValidationError(fmt.Sprintf("provider %s is not supported", name))
		}

		if len(provider.AuthenticationParam) != 0 {
			val, _ := setting.Setting[provider.AuthenticationParam]
			if len(val) == 0 {
				return internal_errors.NewValidationError(fmt.Sprintf("provider %s is missing value for field %s", setting.Provider, provider.AuthenticationParam))
			}
		}
	}

	if setting.Provider == "openai" || setting.Provider == "anthropic" {
		val, _ := setting.Setting["apikey"]
		if len(val) == 0 {
			return internal_errors.NewValidationError("api key is required")
		}
	}

	return nil
}

func (m *ProviderSettingsManager) validateUpdateSettings(providerName string, setting *provider.UpdateSetting) error {
	if providerName != "openai" && providerName != "anthropic" {
		provider, err := m.Storage.GetCustomProviderByName(providerName)
		_, ok := err.(notFoundError)
		if ok {
			return internal_errors.NewValidationError(fmt.Sprintf("provider %s is not supported", providerName))
		}

		if len(provider.AuthenticationParam) != 0 {
			val, _ := setting.Setting[provider.AuthenticationParam]
			if len(val) == 0 {
				return internal_errors.NewValidationError(fmt.Sprintf("provider %s is missing value for field %s", providerName, provider.AuthenticationParam))
			}
		}
	}

	if providerName == "openai" || providerName == "anthropic" {
		val, _ := setting.Setting["apikey"]
		if len(val) == 0 {
			return internal_errors.NewValidationError("api key is required")
		}
	}

	return nil
}

func (m *ProviderSettingsManager) CreateSetting(setting *provider.Setting) (*provider.Setting, error) {
	if len(setting.Provider) == 0 {
		return nil, internal_errors.NewValidationError("provider field cannot be empty")
	}

	if err := m.validateSettings(setting.Provider, setting); err != nil {
		return nil, err
	}

	setting.Id = util.NewUuid()
	setting.CreatedAt = time.Now().Unix()
	setting.UpdatedAt = time.Now().Unix()

	return m.Storage.CreateProviderSetting(setting)
}

func (m *ProviderSettingsManager) UpdateSetting(id string, setting *provider.UpdateSetting) (*provider.Setting, error) {
	if len(id) == 0 {
		return nil, internal_errors.NewValidationError("id cannot be empty")
	}

	existing, _ := m.Storage.GetProviderSetting(id)
	if existing == nil {
		return nil, internal_errors.NewNotFoundError("provider setting is not found")
	}

	if len(setting.Setting) != 0 {
		if err := m.validateUpdateSettings(existing.Provider, setting); err != nil {
			return nil, err
		}
	}

	setting.UpdatedAt = time.Now().Unix()

	return m.Storage.UpdateProviderSetting(id, setting)
}

func (m *ProviderSettingsManager) GetSetting(id string) (*provider.Setting, error) {
	setting := m.MemDb.GetSetting(id)

	if setting == nil {
		return nil, internal_errors.NewNotFoundError("provider setting is not found")
	}

	return setting, nil
}

func (m *ProviderSettingsManager) GetSettings() ([]*provider.Setting, error) {
	settings, err := m.Storage.GetProviderSettings(false)
	if err != nil {
		return nil, internal_errors.NewNotFoundError("provider setting is not found")
	}

	return settings, nil
}
