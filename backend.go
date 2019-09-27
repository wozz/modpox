package modpox

import (
	"fmt"
	"log"
	"net/http"

	"github.com/wozz/modpox/upstream"
)

// Backend is an upstream that can also store data as an intermediate cache
type Backend interface {
	upstream.Upstream

	Put(string, []byte, int) error
}

type noopBackend struct {
	upstream upstream.Upstream
}

func (nb *noopBackend) Get(key string) ([]byte, int, error) {
	return nb.upstream.Get(key)
}

func (nb *noopBackend) Put(key string, val []byte, status int) error {
	return nil
}

type backendCacheUpstream struct {
	upstream upstream.Upstream
	backend  Backend
	cache    *cache
}

func (bcu *backendCacheUpstream) Get(key string) ([]byte, int, error) {
	if b, i := bcu.cache.get(key); b != nil && i != 0 {
		return b, i, nil
	}
	if b, i, err := bcu.backend.Get(key); err == nil {
		return b, i, nil
	} else {
		log.Printf("backend err, fallback to upstream: %s, %v", key, err)
	}
	b, i, err := bcu.upstream.Get(key)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: backendCacheUpstream error", err)
	}
	if i == http.StatusOK {
		bcu.backend.Put(key, b, i)
	} else {
		log.Printf("not adding to backend for non 200 status code: %s, %d", key, i)
	}
	bcu.cache.set(key, b, i)
	return b, i, nil
}

func (bcu *backendCacheUpstream) Put(key string, val []byte, status int) error {
	bcu.cache.set(key, val, status)
	if err := bcu.backend.Put(key, val, status); err != nil {
		log.Printf("backend error: %s, %v", key, err)
	}
	return nil
}
