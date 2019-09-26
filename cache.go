package modpox

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wozz/modpox/upstream"
)

const defaultTTL = time.Hour * 24

type value struct {
	expireTime time.Time
	value      []byte
	status     int
}

type cache struct {
	mu sync.RWMutex
	c  map[string]*value
}

func newCache() *cache {
	c := &cache{
		c: make(map[string]*value),
	}
	go func() {
		t := time.NewTicker(time.Minute)
		for {
			select {
			case <-t.C:
				c.clean()
			}
		}
	}()
	return c
}

func (c *cache) set(key string, val []byte, status int) {
	ttl := defaultTTL
	if strings.HasSuffix(key, "/@latest") ||
		strings.HasSuffix(key, "/@v/list") {
		ttl = time.Hour
	}
	c.setWithTTL(key, val, status, ttl)
}

func (c *cache) setWithTTL(key string, val []byte, status int, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.c[key] = &value{
		expireTime: time.Now().Add(ttl),
		value:      val,
		status:     status,
	}
}

func (c *cache) get(key string) ([]byte, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.c[key]
	if !ok {
		return nil, 0
	}
	if time.Now().After(val.expireTime) {
		return nil, 0
	}
	return val.value, val.status
}

func (c *cache) clean() {
	c.mu.Lock()
	defer c.mu.Unlock()
	keyToDel := []string{}
	for k, v := range c.c {
		if time.Now().After(v.expireTime) {
			keyToDel = append(keyToDel, k)
		}
	}
	for _, k := range keyToDel {
		delete(c.c, k)
	}
}

type cachingUpstream struct {
	cache    *cache
	upstream upstream.Upstream
}

func (cu *cachingUpstream) Get(key string) ([]byte, int, error) {
	if b, i := cu.cache.get(key); b != nil && i != 0 {
		return b, i, nil
	}
	b, i, err := cu.upstream.Get(key)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: cachingUpstream error", err)
	}
	cu.cache.set(key, b, i)
	return b, i, nil
}
