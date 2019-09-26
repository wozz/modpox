package modpox

import (
	"net/http"
	"strings"

	"github.com/wozz/modpox/upstream"
)

type blacklistUpstream struct {
	upstream  upstream.Upstream
	blacklist []string
}

func (bl *blacklistUpstream) Get(key string) ([]byte, int, error) {
	for _, val := range bl.blacklist {
		if strings.HasPrefix(key, val) {
			return nil, http.StatusForbidden, nil
		}
	}
	return bl.upstream.Get(key)
}
