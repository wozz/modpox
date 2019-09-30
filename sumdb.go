package modpox

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/wozz/modpox/upstream"
)

var supportedSumDatabases = [...]string{
	"sum.golang.org",
}

type sumDBUpstream struct {
	upstream upstream.Upstream
}

func (sdb *sumDBUpstream) Get(key string) ([]byte, int, error) {
	if strings.HasPrefix(key, "/sumdb/") {
		if strings.HasSuffix(key, "/supported") {
			for _, db := range supportedSumDatabases {
				if key == fmt.Sprintf("/sumdb/%s/supported", db) {
					return []byte{}, 200, nil
				}
			}
			return []byte{}, 404, nil
		}
		endpoint := ""
		for _, db := range supportedSumDatabases {
			if strings.HasPrefix(key, fmt.Sprintf("/sumdb/%s/", db)) {
				endpoint = db
			}
		}
		if endpoint == "" {
			return []byte{}, 404, nil
		}
		log.Printf("query sumdb: %s %s", endpoint, key)
		keyParts := strings.Split(key, "/")
		if len(keyParts) < 3 {
			return nil, 0, fmt.Errorf("invalid sumdb key: %s", key)
		}
		realPath := strings.Join(keyParts[3:], "/")
		hc := &http.Client{
			Timeout: time.Minute,
		}
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/%s", endpoint, realPath), nil)
		if err != nil {
			log.Printf("could not create http req: %v", err)
			return nil, 0, fmt.Errorf("%w: could not create http req", err)
		}
		req.Header.Set("User-Agent", useragent)
		resp, err := hc.Do(req)
		if err != nil {
			log.Printf("sumdb error: %v", err)
			return nil, 0, fmt.Errorf("%w: sumdb error", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Printf("unexpected status code: %d", resp.StatusCode)
		}
		var b bytes.Buffer
		io.Copy(&b, resp.Body)
		return b.Bytes(), resp.StatusCode, nil
	}
	return sdb.upstream.Get(key)
}
