package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/voxelbrain/goptions"
	"github.com/voxelbrain/kartoffelsack"
)

var (
	options = struct {
		Listen string        `goptions:"-l, --listen, description='Address to bind webserver to'"`
		Maps   []*Map        `goptions:"-m, --map, description='Map a path to a ressource', obligatory"`
		Help   goptions.Help `goptions:"-h, --help, description='Show this help'"`
	}{
		Listen: fmt.Sprintf(":%s", DefaultEnv("PORT", "8080")),
	}
)

func DefaultEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func main() {
	goptions.ParseAndFail(&options)

	for _, m := range options.Maps {
		var h http.Handler
		switch m.Ressource.Scheme {
		case "http", "https":
			h = kartoffelsack.NewSingleHostReverseProxy(m.Ressource)
		default:
			log.Fatalf("Unknown scheme: %s", m.Ressource.Scheme)
		}
		http.Handle(m.Path, http.StripPrefix(m.Path, h))
	}
	log.Printf("Starting webserver on %s...", options.Listen)
	err := http.ListenAndServe(options.Listen, nil)
	if err != nil {
		log.Fatalf("Could not start webserver: %s", err)
	}
}

type Map struct {
	Path      string
	Ressource *url.URL
}

func (m *Map) MarshalGoption(v string) error {
	maps := strings.Split(v, "=>")
	if len(maps) != 2 {
		return fmt.Errorf("Invalid mapping")
	}

	path := strings.TrimSpace(maps[0])
	rsrc, err := url.Parse(strings.TrimSpace(maps[1]))
	if err != nil {
		return fmt.Errorf("Invalid URI: %s", err)
	}

	m.Path = path
	m.Ressource = rsrc
	return nil
}
