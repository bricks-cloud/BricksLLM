package manager

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/util"
)

type ProviderSettingsStorage interface {
	UpdateProviderSetting(id string, setting *provider.UpdateSetting) (*provider.Setting, error)
	CreateProviderSetting(setting *provider.Setting) (*provider.Setting, error)
	GetProviderSetting(id string, withSecret bool) (*provider.Setting, error)
	GetCustomProviderByName(name string) (*custom.Provider, error)
	GetProviderSettings(withSecret bool, ids []string) ([]*provider.Setting, error)
}

type ProviderSettingsCache interface {
	Set(pid string, value any, ttl time.Duration) error
	Get(pid string) (*provider.Setting, error)
	Delete(pid string) error
}

type ProviderSettingsManager struct {
	Storage ProviderSettingsStorage
	Cache   ProviderSettingsCache
}

func NewProviderSettingsManager(s ProviderSettingsStorage, cache ProviderSettingsCache) *ProviderSettingsManager {
	return &ProviderSettingsManager{
		Storage: s,
		Cache:   cache,
	}
}

func isProviderNativelySupported(provider string) bool {
	return provider == "openai" || provider == "anthropic" || provider == "azure" || provider == "vllm" || provider == "deepinfra"
}

func findMissingAuthParams(providerName string, params map[string]string) string {
	missingFields := []string{}

	if providerName == "openai" || providerName == "anthropic" || providerName == "deepinfra" {
		val := params["apikey"]
		if len(val) == 0 {
			missingFields = append(missingFields, "apikey")
		}

		return strings.Join(missingFields, " ,")
	}

	if providerName == "azure" {
		val := params["resourceName"]
		if len(val) == 0 {
			missingFields = append(missingFields, "resourceName")
		}

		val = params["apikey"]
		if len(val) == 0 {
			missingFields = append(missingFields, "apikey")
		}
	}

	if providerName == "vllm" {
		val := params["url"]
		if len(val) == 0 {
			missingFields = append(missingFields, "url")
		}
	}

	return strings.Join(missingFields, ",")
}

func (m *ProviderSettingsManager) validateSettings(providerName string, setting map[string]string) error {
	if !isProviderNativelySupported(providerName) {
		provider, err := m.Storage.GetCustomProviderByName(providerName)
		_, ok := err.(notFoundError)
		if ok {
			return internal_errors.NewValidationError(fmt.Sprintf("provider %s is not supported", providerName))
		}

		if len(provider.AuthenticationParam) != 0 {
			val := setting[provider.AuthenticationParam]
			if len(val) == 0 {
				return internal_errors.NewValidationError(fmt.Sprintf("provider %s is missing value for field %s", providerName, provider.AuthenticationParam))
			}
		}
	}

	missing := findMissingAuthParams(providerName, setting)
	if len(missing) != 0 {
		return internal_errors.NewValidationError(fmt.Sprintf("provider %s is missing fields %s", providerName, missing))
	}

	return nil
}

func (m *ProviderSettingsManager) CreateSetting(setting *provider.Setting) (*provider.Setting, error) {
	if len(setting.Provider) == 0 {
		return nil, internal_errors.NewValidationError("provider field cannot be empty")
	}

	if err := m.validateSettings(setting.Provider, setting.Setting); err != nil {
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

	existing, _ := m.Storage.GetProviderSetting(id, true)
	if existing == nil {
		return nil, internal_errors.NewNotFoundError("provider setting is not found")
	}

	if len(setting.Setting) != 0 {
		if err := m.validateSettings(existing.Provider, setting.Setting); err != nil {
			return nil, err
		}

		merged := existing.Setting
		for k, v := range setting.Setting {
			merged[k] = v
		}

		setting.Setting = merged
	}

	setting.UpdatedAt = time.Now().Unix()

	return m.Storage.UpdateProviderSetting(id, setting)
}

func (m *ProviderSettingsManager) GetSettingViaCache(id string) (*provider.Setting, error) {
	setting, _ := m.Cache.Get(id)
	if setting == nil {
		telemetry.Incr("bricksllm.provider_settings_manager.get_provider_setting.cache_miss", nil, 1)

		stored, err := m.Storage.GetProviderSetting(id, true)
		if err != nil {
			return nil, err
		}

		bs, err := json.Marshal(stored)
		if err != nil {
			return stored, nil
		}

		err = m.Cache.Set(id, bs, time.Hour)
		if err != nil {
			telemetry.Incr("bricksllm.provider_settings_manager.get_setting_via_cache.set_error", nil, 1)
		}

		setting = stored
	}

	if setting != nil {
		telemetry.Incr("bricksllm.provider_settings_manager.get_provider_setting.cache_hit", nil, 1)
	}

	return setting, nil
}

func (m *ProviderSettingsManager) GetSettingsViaCache(ids []string) ([]*provider.Setting, error) {
	settings := []*provider.Setting{}

	for _, id := range ids {
		setting, err := m.GetSettingViaCache(id)
		if err != nil {
			return nil, err
		}

		settings = append(settings, setting)
	}

	return settings, nil
}
