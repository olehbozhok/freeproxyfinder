package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/armon/go-socks5"
	"github.com/gorilla/mux"
)

func checkErr(err error) {
	if err != nil {
		log.Fatalf("error %v", err)
	}
}
func main() {
	addr := ":8235"
	args := os.Args
	if len(args) > 1 {
		addr = args[1]
	}

	proxyW := ProxyWorker{}

	// run now
	go proxyW.UpdateProxies()

	go func() {
		for range time.Tick(time.Minute * 15) {
			proxyW.UpdateProxies()
		}
	}()

	r := mux.NewRouter()
	r.HandleFunc("/", proxyW.HttpHandler).Methods("GET")

	srv := &http.Server{
		Addr: addr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 60,
		ReadTimeout:  time.Second * 60,
		IdleTimeout:  time.Second * 60,
		Handler:      r, // Pass our instance of gorilla/mux in.
	}

	// log.Printf("srv run on %s\n", addr)
	// log.Fatalln(srv.ListenAndServe())
	_ = srv

	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer, err := proxyW.GetDialer()
			if err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}
	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	// Create SOCKS5 proxy on localhost port 8000
	if err := server.ListenAndServe("tcp", ":8000"); err != nil {
		panic(err)
	}

}
