package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/nycmonkey/movers"
	"golang.org/x/crypto/acme/autocert"
)

var (
	fqdn string
	hdlr http.Handler
)

func main() {
	flag.StringVar(&fqdn, "n", `www.example.com`, `domain name for TLS cert`)
	flag.Parse()
	m := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(fqdn),
		Cache:      autocert.DirCache(cacheDir()),
	}
	mux := mux.NewRouter()
	hdlr = movers.NewHandler(mux)
	tlsServer := &http.Server{
		Addr: ":https",
		TLSConfig: &tls.Config{
			GetCertificate: m.GetCertificate,
		},
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      hdlr,
	}
	go func() {
		err := tlsServer.ListenAndServeTLS("", "")
		if err != nil {
			log.Fatalf(`ListenAndServeTLS failed wtih %s`, err)
		}
	}()
	httpServer := &http.Server{
		Addr:         ":http",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      m.HTTPHandler(hdlr),
	}
	err := httpServer.ListenAndServe()
	if err != nil {
		log.Fatalf(`ListenAndServe failed with %s`, err)
	}
}

func cacheDir() (dir string) {
	if u, _ := user.Current(); u != nil {
		dir = filepath.Join(os.TempDir(), "cache-golang-autocert-"+u.Username)
		if err := os.MkdirAll(dir, 0700); err == nil {
			return dir
		}
	}
	return "."
}
