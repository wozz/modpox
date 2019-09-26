package modpox

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/wozz/modpox/upstream"
)

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
}

const useragent = "modpox"

type randomUpstream struct {
	upstreams []upstream.Upstream
}

func (r *randomUpstream) Get(key string) ([]byte, int, error) {
	if r.upstreams == nil || len(r.upstreams) == 0 {
		return nil, 0, fmt.Errorf("no upstreams configured")
	}
	return r.upstreams[rand.Intn(len(r.upstreams))].Get(key)
}

type proxyUpstream struct {
	endpoint string
}

func (p *proxyUpstream) Get(key string) ([]byte, int, error) {
	log.Printf("query upstream: %s %s", p.endpoint, key)
	hc := &http.Client{
		Timeout: time.Minute,
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", p.endpoint, key), nil)
	if err != nil {
		log.Printf("could not create http req: %v", err)
		return nil, 0, fmt.Errorf("%w: could not create http req", err)
	}
	req.Header.Set("User-Agent", useragent)
	resp, err := hc.Do(req)
	if err != nil {
		log.Printf("upstream error: %v", err)
		return nil, 0, fmt.Errorf("%w: could not perform req", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("unexpected status code: %d", resp.StatusCode)
	}
	var b bytes.Buffer
	io.Copy(&b, resp.Body)
	return b.Bytes(), resp.StatusCode, nil
}
