package manager

import (
	"errors"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/util"
)

type Storage interface {
	GetKeys(tags, keyIds []string, provider string) ([]*key.ResponseKey, error)
	UpdateKey(id string, key *key.UpdateKey) (*key.ResponseKey, error)
	CreateKey(key *key.RequestKey) (*key.ResponseKey, error)
	DeleteKey(id string) error
	GetProviderSetting(id string) (*provider.Setting, error)
	GetProviderSettings(withSecret bool, ids []string) ([]*provider.Setting, error)
}

type Encrypter interface {
	Encrypt(secret string) string
}

type Manager struct {
	s Storage
	e Encrypter
}

func NewManager(s Storage, e Encrypter) *Manager {
	return &Manager{
		s: s,
		e: e,
	}
}

func (m *Manager) GetKeys(tags, keyIds []string, provider string) ([]*key.ResponseKey, error) {
	return m.s.GetKeys(tags, keyIds, provider)
}

func (m *Manager) CreateKey(rk *key.RequestKey) (*key.ResponseKey, error) {
	rk.CreatedAt = time.Now().Unix()
	rk.UpdatedAt = time.Now().Unix()
	rk.Key = m.e.Encrypt(rk.Key)
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
	}

	return m.s.UpdateKey(id, uk)
}

func (m *Manager) DeleteKey(id string) error {
	return m.s.DeleteKey(id)
}
