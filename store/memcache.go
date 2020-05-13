package store

import (
	"encoding/json"
	"io"
	"sync"
)

type Cacher interface {
	Get(k string) (string, bool)
	Set(k, v string) error
	Del(k string)
	Marshal() ([]byte, error)
	UnMarshal(serialized io.ReadCloser) error
}

type cache struct {
	mtx sync.RWMutex
	kv  map[string]string
}

func NewCache() Cacher {
	return &cache{
		kv: make(map[string]string),
	}
}
func (c *cache) Set(k, v string) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.kv[k] = v
	return nil
}

func (c *cache) Get(k string) (string, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	val, ok := c.kv[k]
	return val, ok
}
func (c *cache) Del(k string) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	delete(c.kv, k)
}

func (c *cache) Marshal() ([]byte, error) {
	c.mtx.RLock()
	c.mtx.RUnlock()
	return json.Marshal(c.kv)
}

func (c *cache) UnMarshal(serialized io.ReadCloser) error {
	var newData map[string]string
	if err := json.NewDecoder(serialized).Decode(&newData); err != nil {
		return err
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.kv = newData
	return nil
}
