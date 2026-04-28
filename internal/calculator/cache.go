package calculator

import "sync"

const maxCacheEntries = 1024

// packCache is a bounded, thread-safe cache for pack calculation results.
// When full it evicts one arbitrary entry before inserting the new one.
type packCache struct {
	mu      sync.Mutex
	entries map[string]map[int]int
	maxSize int
}

func newPackCache(maxSize int) *packCache {
	return &packCache{
		entries: make(map[string]map[int]int, maxSize),
		maxSize: maxSize,
	}
}

func (c *packCache) get(key string) (map[int]int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	return copyPacks(v), true
}

func (c *packCache) set(key string, value map[int]int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxSize {
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}
	c.entries[key] = copyPacks(value)
}

func copyPacks(m map[int]int) map[int]int {
	out := make(map[int]int, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
