package storage

import "github.com/bricks-cloud/bricksllm/internal/key"

type Storage interface {
	GetKeys(tag string) ([]*key.ResponseKey, error)
	UpdateKey(id string, key *key.RequestKey) (*key.ResponseKey, error)
	CreateKey(id string, key *key.RequestKey) (*key.ResponseKey, error)
	DeleteKey(id string) error
}
