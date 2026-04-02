package cache

import (
	"strconv"
	"sync"

	"golang.org/x/sync/singleflight"
)

const VersionContentCacheMaxEntries = 500

type VersionContentCache struct {
	mu    sync.RWMutex
	sf    singleflight.Group
	cache map[int][]byte
	order []int
}

func NewVersionContentCache() *VersionContentCache {
	return &VersionContentCache{cache: make(map[int][]byte)}
}

func (c *VersionContentCache) Get(id int) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	b, ok := c.cache[id]
	if !ok {
		return nil, false
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out, true
}

func (c *VersionContentCache) Set(id int, content []byte) {
	if len(content) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	clone := make([]byte, len(content))
	copy(clone, content)
	if _, exists := c.cache[id]; exists {
		c.cache[id] = clone
		return
	}
	for len(c.order) >= VersionContentCacheMaxEntries {
		evict := c.order[0]
		c.order = c.order[1:]
		delete(c.cache, evict)
	}
	c.cache[id] = clone
	c.order = append(c.order, id)
}

func (c *VersionContentCache) GetOrLoad(id int, load func() ([]byte, error)) ([]byte, error) {
	key := strconv.Itoa(id)
	v, err, _ := c.sf.Do(key, func() (interface{}, error) {
		if b, ok := c.Get(id); ok {
			return b, nil
		}
		content, err := load()
		if err != nil {
			return nil, err
		}
		c.Set(id, content)
		return content, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}

var GlobalContentCache = NewVersionContentCache()
