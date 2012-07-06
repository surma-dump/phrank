package main

import (
	"code.google.com/p/gorilla/mux"
	"flag"
	"fmt"
	"net/http"
)

var (
	help = flag.Bool("h", false, "Show this help")
)

func main() {
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		return
	}

	httpr := mux.NewRouter()
	httpr.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+r.Host, http.StatusMovedPermanently)
	})
	go http.ListenAndServe("0.0.0.0:80", httpr)

	httpsr := mux.NewRouter()
	httpsr.PathPrefix("/").HandlerFunc(handler)
	e := http.ListenAndServeTLS("0.0.0.0:443", "cert.pem", "key.pem", httpsr)
	if e != nil {
		panic(e)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hai")
}
