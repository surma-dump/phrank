package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"

	"github.com/voxelbrain/goptions"
	"github.com/voxelbrain/katalysator"
)

var (
	options = struct {
		Listen        string        `goptions:"-l, --listen, description='Address to bind webserver to'"`
		CacheDuration time.Duration `goptions:"-c, --cache-duration, description='Duration to cache static content'"`
		Maps          []*Map        `goptions:"-m, --map, description='Map a path to a ressource', obligatory"`
		Help          goptions.Help `goptions:"-h, --help, description='Show this help'"`
	}{
		Listen:        fmt.Sprintf(":%s", DefaultEnv("PORT", "8080")),
		CacheDuration: 5 * time.Minute,
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
		if !strings.HasSuffix(m.Path, "/") {
			m.Path += "/"
		}
		var h http.Handler
		switch m.Ressource.Scheme {
		case "http", "https":
			h = katalysator.NewSingleHostReverseProxy(m.Ressource)
		case "file":
			h = http.FileServer(http.Dir(m.Ressource.Path))
			h = katalysator.NewCache(options.CacheDuration, h)
		case "s3":
			username := m.Ressource.User.Username()
			password, _ := m.Ressource.User.Password()
			elems := strings.Split(strings.TrimLeft(m.Ressource.Path, "/"), "/")
			bucketname := elems[0]
			prefix := "/"
			if len(elems) > 1 {
				prefix += path.Join(elems[1:]...)
			}
			region, err := regionByEndpoint(m.Ressource.Host)
			if err != nil {
				log.Fatalf("Could not connect to S3: %s", err)
			}
			auth := aws.Auth{
				AccessKey: username,
				SecretKey: password,
			}
			s3acc := s3.New(auth, region)
			h = &S3HTTP{
				Bucket: s3acc.Bucket(bucketname),
				Prefix: prefix,
			}
			h = katalysator.NewCache(options.CacheDuration, h)
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

func regionByEndpoint(endpoint string) (aws.Region, error) {
	for _, region := range aws.Regions {
		epURL, _ := url.Parse(region.S3Endpoint)
		if epURL.Host == endpoint {
			return region, nil
		}
	}
	return aws.Region{}, fmt.Errorf("No region with endpoint %s", endpoint)
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

type S3HTTP struct {
	Bucket *s3.Bucket
	Prefix string
}

func (s *S3HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := path.Join(s.Prefix, r.URL.Path)[1:]
	resp, err := s.Bucket.List(key, "", "", 2)
	if err != nil {
		log.Printf("Could not list bucket %s: %s", s.Bucket.Name, err)
		http.NotFound(w, r)
		return
	}
	if len(resp.Contents) != 1 {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", resp.Contents[0].Size))
	w.Header().Set("ETag", resp.Contents[0].ETag)
	f, _ := s.Bucket.GetReader(key)
	defer f.Close()

	io.Copy(w, f)
}
