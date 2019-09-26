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

type sumDBUpstream struct {
	upstream upstream.Upstream
	endpoint string
}

func (sdb *sumDBUpstream) Get(key string) ([]byte, int, error) {
	if strings.HasPrefix(key, "/sumdb/") {
		// ensure the client knows to use this sumdb
		if strings.HasSuffix(key, "/supported") {
			return []byte{}, 200, nil
		}
		log.Printf("query sumdb: %s %s", sdb.endpoint, key)
		keyParts := strings.Split(key, "/")
		if len(keyParts) < 3 {
			return nil, 0, fmt.Errorf("invalid sumdb key: %s", key)
		}
		realPath := strings.Join(keyParts[3:], "/")
		hc := &http.Client{
			Timeout: time.Minute,
		}
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/%s", sdb.endpoint, realPath), nil)
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
