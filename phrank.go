package main

import (
	"bufio"
	"code.google.com/p/gorilla/mux"
	"encoding/json"
	"flag"
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
	httpAddr  = flag.String("http", ":80", "Address to bind HTTP server to")
	httpsAddr = flag.String("https", ":443", "Address to bind HTTP server to")
	configDir = flag.String("config", "phrank.d", "Location of the config directory")
	help      = flag.Bool("h", false, "Show this help")
)

func main() {
	flag.Parse()

	if *help {
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

	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(appHandler)
	log.Printf("Binding HTTPS to %s", *httpsAddr)
	go func() {
		e := http.ListenAndServeTLS(*httpsAddr, *configDir+"/cert.pem", *configDir+"/key.pem", router)
		if e != nil {
			log.Printf("Could not bind HTTPS server: %s", e)
		}
	}()

	log.Printf("Binding HTTP to %s", *httpAddr)
	e := http.ListenAndServe(*httpAddr, router)
	if e != nil {
		log.Fatalf("Could not bind HTTP server: %s", e)
	}
}

var (
	backends = map[string]Backend{}
)

type Backend struct {
	Domain           string
	AddForwardHeader bool
	Address          string
}

func readConfig() {
	log.Printf("Reading configuration files...")
	newBackends := map[string]Backend{}
	root := *configDir+"/apps"
	filepath.Walk(root, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			log.Printf("Error: %s: %s", path, e)
			return nil
		}
		if info.IsDir() && path != root {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(path, ".app") {
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
		newBackends[strings.ToLower(b.Domain)] = b
		return nil
	})
	backends = newBackends
	log.Printf("Done.")
}

func appHandler(w http.ResponseWriter, r *http.Request) {
	backend, ok := backends[strings.ToLower(r.Host)]
	if !ok {
		http.Error(w, "Invalid domain", 404)
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
