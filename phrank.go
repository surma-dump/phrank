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
	"github.com/voxelbrain/k"
)

const (
	VERSION = "1.0.1"
)

var (
	options = struct {
		Listen        string        `goptions:"-l, --listen, description='Address to bind webserver to'"`
		CacheDuration time.Duration `goptions:"-c, --cache-duration, description='Duration to cache static content'"`
		Maps          []*Map        `goptions:"-m, --map, description='Map a path to a resource', obligatory"`
		Help          goptions.Help `goptions:"-h, --help, description='Show this help'"`
	}{
		Listen: fmt.Sprintf(":%s", k.DefaultEnv("PORT", "8080")),
	}
)

func main() {
	goptions.ParseAndFail(&options)

	for _, m := range options.Maps {
		h := m.Handler
		if m.Cachable {
			h = k.NewCache(options.CacheDuration, m.Handler)
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

func splitS3URL(u *url.URL) (*s3.Bucket, string, error) {
	username := u.User.Username()
	password, _ := u.User.Password()
	elems := strings.Split(strings.TrimLeft(u.Path, "/"), "/")
	bucketname := elems[0]
	prefix := "/"
	if len(elems) > 1 {
		prefix += path.Join(elems[1:]...)
	}
	region, err := regionByEndpoint(u.Host)
	if err != nil {
		return nil, "", err
	}
	auth := aws.Auth{
		AccessKey: username,
		SecretKey: password,
	}
	s3acc := s3.New(auth, region)
	return s3acc.Bucket(bucketname), prefix, nil
}

type Map struct {
	Path     string
	Handler  http.Handler
	Cachable bool
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
		m.Cachable = false
		m.Handler = k.NewSingleHostReverseProxy(resource)
	case "file":
		m.Cachable = true
		m.Handler = http.FileServer(http.Dir(resource.Path))
	case "s3":
		m.Cachable = true
		bucket, prefix, err := splitS3URL(resource)
		if err != nil {
			return fmt.Errorf("S3-related error: %s", err)
		}
		m.Handler = &S3HTTP{
			Bucket: bucket,
			Prefix: prefix,
		}
	default:
		return fmt.Errorf("Unsupported scheme: %s", resource.Scheme)
	}
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
