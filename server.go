package modpox

import (
	"log"
	"net/http"

	"github.com/wozz/modpox/upstream"
)

const (
	upstreamEndpoint = "https://proxy.golang.org"
)

var (
	blacklist = []string{}
	token     = ""
)

// SetToken sets a token to be used for private gitlab API interaction
func SetToken(t string) {
	token = t
}

// Server is the main go mod proxy
type Server struct {
	srv     *http.Server
	backend Backend
}

func newHandler(srv *Server) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("req: %s", r.URL.Path)
		data, status, err := srv.backend.Get(r.URL.Path)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("backend error: %v", err)
			return
		}
		w.WriteHeader(status)
		w.Write(data)
	}
}

// NewServer creates a new Server with default settings
// This will likely change to be more configurable in the future
func NewServer() *Server {
	localCache := newCache()
	upstream1 := &blacklistUpstream{
		blacklist: blacklist,
		upstream: &cachingUpstream{
			cache: localCache,
			upstream: &sumDBUpstream{
				endpoint: "sum.golang.org",
				upstream: &randomUpstream{
					upstreams: []upstream.Upstream{
						&proxyUpstream{endpoint: upstreamEndpoint},
					},
				},
			},
		},
	}
	/* gitlab upstream
	upstream2 := gitlab.NewGitLabUpstream(&gitlab.Config{
		Host:     "<gitlab host>",
		Token:    token,
		Upstream: upstream1,
	})
	*/
	backend := &noopBackend{
		upstream: upstream1,
	}
	s := &Server{
		srv: &http.Server{
			Addr: "127.0.0.1:8080",
		},
		backend: backend,
	}
	http.HandleFunc("/", newHandler(s))
	return s
}

// Start starts the server asyncronously and returns immediately
func (s *Server) Start() {
	go func() {
		if err := s.srv.ListenAndServe(); err != nil {
			log.Printf("http server error: %v", err)
		}
	}()
}
