package manager

import (
	"github.com/bricks-cloud/bricksllm/internal/key"
)

type Storage interface {
	GetKeysByTag(tag string) ([]*key.ResponseKey, error)
	UpdateKey(id string, key *key.RequestKey) (*key.ResponseKey, error)
	CreateKey(key *key.RequestKey) (*key.ResponseKey, error)
	DeleteKey(id string) error
}

type Manager struct {
	s Storage
}

func NewManager(s Storage) *Manager {
	return &Manager{
		s: s,
	}
}

func (m *Manager) GetKeysByTag(tag string) ([]*key.ResponseKey, error) {
	return m.s.GetKeysByTag(tag)
}

func (m *Manager) CreateKey(rk *key.RequestKey) (*key.ResponseKey, error) {
	if err := rk.Validate(); err != nil {
		return nil, err
	}

	return m.s.CreateKey(rk)
}

func (m *Manager) UpdateKey(id string, rk *key.RequestKey) (*key.ResponseKey, error) {
	return m.s.UpdateKey(id, rk)
}

func (m *Manager) DeleteKey(id string) error {
	return m.s.DeleteKey(id)
}
