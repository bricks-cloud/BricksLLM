package manager

import (
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/util"
)

type Storage interface {
	GetKeysByTag(tag string) ([]*key.ResponseKey, error)
	UpdateKey(id string, key *key.UpdateKey) (*key.ResponseKey, error)
	CreateKey(key *key.RequestKey) (*key.ResponseKey, error)
	DeleteKey(id string) error
	GetProviderSetting(id string) (*provider.Setting, error)
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

func (m *Manager) GetKeysByTag(tag string) ([]*key.ResponseKey, error) {
	return m.s.GetKeysByTag(tag)
}

func (m *Manager) CreateKey(rk *key.RequestKey) (*key.ResponseKey, error) {
	rk.CreatedAt = time.Now().Unix()
	rk.UpdatedAt = time.Now().Unix()
	rk.Key = m.e.Encrypt(rk.Key)
	rk.KeyId = util.NewUuid()

	if err := rk.Validate(); err != nil {
		return nil, err
	}

	if _, err := m.s.GetProviderSetting(rk.SettingId); err != nil {
		return nil, err
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

	return m.s.UpdateKey(id, uk)
}

func (m *Manager) DeleteKey(id string) error {
	return m.s.DeleteKey(id)
}
