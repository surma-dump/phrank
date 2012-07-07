package main

import (
	"code.google.com/p/gorilla/mux"
	"flag"
	"fmt"
	"log"
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
	go http.ListenAndServe(httpAddr, httpr)

	httpsr := mux.NewRouter()
	httpsr.PathPrefix("/").HandlerFunc(handler)
	log.Printf("Binding HTTPS to %s", httpsAddr)
	e := http.ListenAndServeTLS(httpsAddr, "cert.pem", "key.pem", httpsr)
	if e != nil {
		log.Fatalf("Could not start webserver: %s", e)
	}
}

var (
	backends = map[string]string{}
)

func handler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.Host, *domainSuffix) {
		http.Error(w, "Invalid domain", 404)
		return
	}
	// Strip domain suffix + dot
	appname := r.Host[0 : len(r.Host)-len(*domainSuffix)-1]
	_, ok := backends[appname]
	if !ok {
		http.Error(w, "Unknown app name", 404)
		return
	}
	fmt.Fprintf(w, "Bla")
}
