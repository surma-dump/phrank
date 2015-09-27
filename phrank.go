package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/voxelbrain/goptions"
)

const (
	VERSION = "2.0.0"
)

var (
	options = struct {
		Listen string        `goptions:"-l, --listen, description='Address to bind webserver to'"`
		Maps   []*Map        `goptions:"-m, --map, description='Map a path to a resource', obligatory"`
		Help   goptions.Help `goptions:"-h, --help, description='Show this help'"`
	}{
		Listen: fmt.Sprintf(":%s", k.DefaultEnv("PORT", "8080")),
	}
)

func main() {
	goptions.ParseAndFail(&options)

	for _, m := range options.Maps {
		h := m.Handler
		http.Handle(m.Path, http.StripPrefix(m.Path, h))
	}
	log.Printf("Starting webserver on %s...", options.Listen)
	err := http.ListenAndServe(options.Listen, nil)
	if err != nil {
		log.Fatalf("Could not start webserver: %s", err)
	}
}

type Map struct {
	Path    string
	Handler http.Handler
}

func (m *Map) MarshalGoption(v string) error {
	maps := strings.Split(v, "=>")
	if len(maps) != 2 {
		return fmt.Errorf("Invalid mapping")
	}

	path := strings.TrimSpace(maps[0])
	resource, err := url.Parse(strings.TrimSpace(maps[1]))
	if err != nil {
		return fmt.Errorf("Invalid URI: %s", err)
	}

	m.Path = path
	if !strings.HasSuffix(m.Path, "/") {
		m.Path += "/"
	}

	switch resource.Scheme {
	case "http", "https":
		m.Handler = NewSingleHostReverseProxy(resource)
	case "file":
		m.Handler = http.FileServer(http.Dir(resource.Path))
	default:
		return fmt.Errorf("Unsupported scheme: %s", resource.Scheme)
	}
	return nil
}

// NewSingleHostReverseProxy is an augmentation of
// `net/http/httputil.NewSingleHostReverseProxy` which additionally sets
// the `Host` header.
func NewSingleHostReverseProxy(url *url.URL) *httputil.ReverseProxy {
	rp := httputil.NewSingleHostReverseProxy(url)
	oldDirector := rp.Director
	rp.Director = func(r *http.Request) {
		oldDirector(r)
		r.Host = url.Host
	}
	return rp
}
