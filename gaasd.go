package main

import (
	"bufio"
	"code.google.com/p/gorilla/mux"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

var (
	httpPort     = flag.Int("http", 80, "Port to bind HTTP server to")
	httpsPort    = flag.Int("https", 443, "Port to bind HTTP server to")
	domainSuffix = flag.String("domain", "", "Domain-suffix for apps")
	configDir    = flag.String("config", "gaas.d", "Location of the config directory")
	help         = flag.Bool("h", false, "Show this help")
)

func main() {
	flag.Parse()

	if *help || *domainSuffix == "" {
		flag.PrintDefaults()
		return
	}

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGUSR1)
		for _ = range c {
			readConfig()
		}
	}()
	readConfig()

	httpAddr := fmt.Sprintf("0.0.0.0:%d", *httpPort)
	httpsAddr := fmt.Sprintf("0.0.0.0:%d", *httpsPort)

	httpr := mux.NewRouter()
	httpr.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, fmt.Sprintf("https://%s:%d", r.Host, *httpsPort), http.StatusMovedPermanently)
	})
	log.Printf("Binding HTTP to %s", httpAddr)
	go func() {
		e := http.ListenAndServe(httpAddr, httpr)
		if e != nil {
			log.Fatalf("Could not bind HTTP server: %s", e)
		}
	}()

	httpsr := mux.NewRouter()
	httpsr.PathPrefix("/").HandlerFunc(appHandler)
	log.Printf("Binding HTTPS to %s", httpsAddr)
	e := http.ListenAndServeTLS(httpsAddr, "cert.pem", "key.pem", httpsr)
	if e != nil {
		log.Fatalf("Could not bind HTTPS server: %s", e)
	}
}

var (
	backends = map[string]Backend{}
)

type Backend struct {
	Name             string
	AddForwardHeader bool
	Address          string
}

func readConfig() {
	log.Printf("Reading configuration files...")
	newBackends := map[string]Backend{}
	filepath.Walk(*configDir, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			log.Printf("Error: %s: %s", path, e)
			return nil
		}
		if info.IsDir() && path != *configDir {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(path, ".conf") {
			return nil
		}
		log.Printf("Reading %s...", path)
		f, e := os.Open(path)
		if e != nil {
			log.Printf("Could not open file %s: %s", path, e)
			return nil
		}
		defer f.Close()

		var b Backend
		dec := json.NewDecoder(f)
		if e := dec.Decode(&b); e != nil {
			log.Printf("Could not parse file %s: %s", path, e)
			return nil
		}
		newBackends[strings.ToLower(b.Name)] = b
		return nil
	})
	backends = newBackends
	log.Printf("Done.")
}

func appHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.Host, *domainSuffix) {
		http.Error(w, "Invalid domain", 404)
		return
	}
	// Strip domain suffix + leading dot
	appname := r.Host[0 : len(r.Host)-len(*domainSuffix)-1]
	backend, ok := backends[appname]
	if !ok {
		http.Error(w, "Unknown app name", 404)
		return
	}

	backendaddr, e := net.ResolveTCPAddr("tcp4", backend.Address)
	if e != nil {
		http.Error(w, "Could not resolve backend address", 500)
		return
	}

	ccon, _, e := w.(http.Hijacker).Hijack()
	if e != nil {
		http.Error(w, "Hijacking failed", 500)
		return
	}
	defer ccon.Close()

	bcon, e := net.DialTCP("tcp4", nil, backendaddr)
	if e != nil {
		log.Printf("Backend not reachable: %s", e)
		return
	}
	defer bcon.Close()

	// Add X-Forwared-For header and send request
	if backend.AddForwardHeader {
		r.Header.Add("X-Forwarded-For", r.RemoteAddr)
	}
	r.Write(bcon)

	// Avoid floating keep-alive connection by letting the http package
	// parse the response. It closes the connection once everything has been
	// received.
	resp, e := http.ReadResponse(bufio.NewReader(bcon), r)
	if e != nil {
		log.Printf("Invalid response: %s", e)
		return
	}
	resp.Write(ccon)
}
