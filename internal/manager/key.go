package manager

import (
	"errors"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/encrypter"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/util"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
)

type Storage interface {
	GetKeys(tags, keyIds []string, provider string) ([]*key.ResponseKey, error)
	UpdateKey(id string, key *key.UpdateKey) (*key.ResponseKey, error)
	CreateKey(key *key.RequestKey) (*key.ResponseKey, error)
	DeleteKey(id string) error
	GetProviderSetting(id string) (*provider.Setting, error)
	GetProviderSettings(withSecret bool, ids []string) ([]*provider.Setting, error)
	GetKey(keyId string) (*key.ResponseKey, error)
}

type costLimitCache interface {
	Delete(keyId string) error
}

type rateLimitCache interface {
	Delete(keyId string) error
}

type accessCache interface {
	Delete(keyId string) error
}

type Encrypter interface {
	Encrypt(secret string) string
}

type Manager struct {
	s   Storage
	clc costLimitCache
	rlc rateLimitCache
	ac  accessCache
}

func NewManager(s Storage, clc costLimitCache, rlc rateLimitCache, ac accessCache) *Manager {
	return &Manager{
		s:   s,
		clc: clc,
		rlc: rlc,
		ac:  ac,
	}
}

func (m *Manager) GetKeys(tags, keyIds []string, provider string) ([]*key.ResponseKey, error) {
	return m.s.GetKeys(tags, keyIds, provider)
}

func (m *Manager) areProviderSettingsUniqueness(settings []*provider.Setting) bool {
	providerMap := map[string]bool{}

	for _, setting := range settings {
		if providerMap[setting.Provider] {
			return false
		}

		providerMap[setting.Provider] = true
	}

	return true
}

func (m *Manager) CreateKey(rk *key.RequestKey) (*key.ResponseKey, error) {
	rk.CreatedAt = time.Now().Unix()
	rk.UpdatedAt = time.Now().Unix()
	rk.Key = encrypter.Encrypt(rk.Key)
	rk.KeyId = util.NewUuid()

	if err := rk.Validate(); err != nil {
		return nil, err
	}

	if len(rk.SettingId) != 0 {
		if _, err := m.s.GetProviderSetting(rk.SettingId); err != nil {
			return nil, err
		}
	}

	if len(rk.SettingIds) != 0 {
		existing, err := m.s.GetProviderSettings(false, rk.SettingIds)
		if err != nil {
			return nil, err
		}

		if len(existing) == 0 {
			return nil, errors.New("provider settings not found")
		}

		if !m.areProviderSettingsUniqueness(existing) {
			return nil, internal_errors.NewValidationError("key can only be assoicated with one setting per provider")
		}

	}

	return m.s.CreateKey(rk)
}

func (m *Manager) UpdateKey(id string, uk *key.UpdateKey) (*key.ResponseKey, error) {
	uk.UpdatedAt = time.Now().Unix()

	if err := uk.Validate(); err != nil {
		return nil, err
	}

	if len(uk.SettingId) != 0 {
		if _, err := m.s.GetProviderSetting(uk.SettingId); err != nil {
			return nil, err
		}
	}

	if len(uk.SettingIds) != 0 {
		existing, err := m.s.GetProviderSettings(false, uk.SettingIds)
		if err != nil {
			return nil, err
		}

		if len(existing) == 0 {
			return nil, errors.New("provider settings not found")
		}

		if !m.areProviderSettingsUniqueness(existing) {
			return nil, internal_errors.NewValidationError("key can only be assoicated with one setting per provider")
		}
	}

	if len(uk.CostLimitInUsdUnit) != 0 {
		err := m.clc.Delete(id)
		if err != nil {
			return nil, err
		}
	}

	if len(uk.RateLimitUnit) != 0 {
		err := m.rlc.Delete(id)
		if err != nil {
			return nil, err
		}
	}

	if len(uk.CostLimitInUsdUnit) != 0 || len(uk.RateLimitUnit) != 0 {
		err := m.ac.Delete(id)
		if err != nil {
			return nil, err
		}
	}

	return m.s.UpdateKey(id, uk)
}

func (m *Manager) DeleteKey(id string) error {
	return m.s.DeleteKey(id)
}
