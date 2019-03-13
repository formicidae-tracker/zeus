package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
	"github.com/grandcat/zeroconf"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Address string `long:"http-listen" short:"l" description:"Address for the HTTP server" default:":3000"`
	RPC     int    `long:"rpc-listen" short:"r" description:"Port for the RPC Service" default:"3001"`
	NoAvahi bool   `long:"no-avahi" short:"n" description:"Do not use avahi service broadcast"`
}

func JSON(w http.ResponseWriter, obj interface{}) {
	data, err := json.Marshal(obj)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func RecoverWrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		defer func() {
			r := recover()
			if r != nil {
				switch t := r.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = t
				default:
					err = errors.New("Unknown error")
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func HTTPLogWrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[http] %s %s from %s %s", r.Method, r.RequestURI, r.RemoteAddr, r.UserAgent())
		h.ServeHTTP(w, r)
	})
}

func runServer(srv *http.Server, wg *sync.WaitGroup, logPrefix string) {
	defer wg.Done()
	idleConnections := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("[%s] Could not shutdown: %v", logPrefix, err)
		}
		close(idleConnections)
	}()
	log.Printf("[%s] listening on %s", logPrefix, srv.Addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("[%s] Could not serve: %v", logPrefix, err)
	}
	<-idleConnections
}

func Execute() error {
	opts := Options{}

	if _, err := flags.Parse(&opts); err != nil {
		return err
	}

	h := NewHermes()

	router := mux.NewRouter()

	router.HandleFunc("/api/zones", func(w http.ResponseWriter, r *http.Request) {
		res := h.getZones()
		JSON(w, &res)
	}).Methods("GET")

	router.HandleFunc("/api/host/{hname}/zone/{zname}/climate-report", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		res, err := h.getClimateReport(vars["hname"], vars["zname"], r.URL.Query().Get("window"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		JSON(w, &res)
	}).Methods("GET")

	router.HandleFunc("/api/host/{hname}/zone/{zname}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		res, err := h.getZone(vars["hname"], vars["zname"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		JSON(w, &res)
	}).Methods("GET")

	angularPaths := []string{
		"/host/{h}/zone/{z}",
	}

	angularAssetsPath := "./webapp/dist/webapp"
	for _, p := range angularPaths {
		router.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			indexBytes, err := ioutil.ReadFile(filepath.Join(angularAssetsPath, "index.html"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			w.Write(indexBytes)
		}).Methods("GET")
	}

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./webapp/dist/webapp")))

	router.Use(HTTPLogWrap)
	router.Use(RecoverWrap)

	wg := sync.WaitGroup{}

	httpServer := http.Server{
		Addr:    opts.Address,
		Handler: router,
	}

	wg.Add(1)
	go runServer(&httpServer, &wg, "http")
	rpcRouter := rpc.NewServer()
	rpcRouter.Register(h)
	rpcRouter.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	rpcServer := http.Server{
		Addr:    fmt.Sprintf(":%d", opts.RPC),
		Handler: rpcRouter,
	}

	wg.Add(1)
	go runServer(&rpcServer, &wg, "rpc")

	if opts.NoAvahi == false {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("Using zeroconf discovery service")
			server, err := zeroconf.Register("Olympus", "_olympus._tcp", "local.", opts.RPC, nil, nil)
			if err != nil {
				log.Printf("[avahi] register error: %s", err)
				return
			}
			sigint := make(chan os.Signal, 1)
			signal.Notify(sigint, os.Interrupt)
			<-sigint
			server.Shutdown()
		}()
	}

	wg.Wait()
	return nil
}

func main() {
	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
