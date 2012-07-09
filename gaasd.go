package main

import (
	"code.google.com/p/gorilla/mux"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

var (
	httpPort     = flag.Int("http", 80, "Port to bind HTTP server to")
	httpsPort    = flag.Int("https", 443, "Port to bind HTTP server to")
	domainSuffix = flag.String("domain", "", "Domain-suffix for apps")
	help         = flag.Bool("h", false, "Show this help")
)

func main() {
	flag.Parse()

	if *help || *domainSuffix == "" {
		flag.PrintDefaults()
		return
	}

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
	backends = map[string]*net.TCPAddr{}
)

func init() {
	var e error
	backends["testapp"], e = net.ResolveTCPAddr("tcp4", "localhost:9001")
	if e != nil {
		log.Fatalf("Invalid app address: %s", e)
	}
}

func appHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.Host, *domainSuffix) {
		http.Error(w, "Invalid domain", 404)
		return
	}
	// Strip domain suffix + dot
	appname := r.Host[0 : len(r.Host)-len(*domainSuffix)-1]
	backend, ok := backends[appname]
	if !ok {
		http.Error(w, "Unknown app name", 404)
		return
	}
	ccon, _, e := w.(http.Hijacker).Hijack()
	if e != nil {
		http.Error(w, "Hijacking failed", 500)
		return
	}
	defer ccon.Close()

	bcon, e := net.DialTCP("tcp4", nil, backend)
	if e != nil {
		log.Printf("Backend not reachable: %s", e)
		return
	}
	defer bcon.Close()

	r.Header.Add("X-Forwarded-For", r.RemoteAddr)
	log.Printf("Forwarding request from %s to %s", r.RemoteAddr, backend)
	r.Write(bcon)

	log.Printf("Waiting for answer...")
	io.Copy(ccon, bcon)
	log.Printf("Done")
}
