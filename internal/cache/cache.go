package cache

import (
	"time"

	"github.com/bricks-cloud/bricksllm/internal/encrypter"
)

type store interface {
	Set(key string, value interface{}, ttl time.Duration) error
	GetBytes(key string) ([]byte, error)
}

type Cache struct {
	store store
}

func NewCache(s store) *Cache {
	return &Cache{
		store: s,
	}
}

func (c *Cache) computeHashKey(value string) string {
	return encrypter.Encrypt(value)
}

func (c *Cache) StoreBytes(key string, value []byte, ttl time.Duration) error {
	return c.store.Set(c.computeHashKey(key), value, ttl)
}

func (c *Cache) GetBytes(key string) ([]byte, error) {
	return c.store.GetBytes(c.computeHashKey(key))
}
