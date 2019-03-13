package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

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

func LogWrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s %s", r.Method, r.RequestURI, r.RemoteAddr, r.UserAgent())
		h.ServeHTTP(w, r)
	})
}

func Execute() error {
	setClimateReporterStub()
	router := mux.NewRouter()

	router.HandleFunc("/api/zones", func(w http.ResponseWriter, r *http.Request) {
		res := []RegisteredZone{
			{Host: "helms-deep", Name: "box"},
			{Host: "helms-deep", Name: "tunnel"},
			{Host: "minas-tirith", Name: "box"},
			{Host: "rivendel", Name: "box"},
		}

		JSON(w, &res)
	}).Methods("GET")

	router.HandleFunc("/api/host/{hname}/zone/{zname}/climate-report", func(w http.ResponseWriter, r *http.Request) {
		askedWindow := r.URL.Query().Get("window")
		var res ClimateReportTimeSerie
		switch askedWindow {
		case "hour":
			res = stubClimateReporter.LastHour()
		case "day":
			res = stubClimateReporter.LastDay()
		case "week":
			res = stubClimateReporter.LastWeek()
		default:
			res = stubClimateReporter.LastDay()
		}
		JSON(w, &res)
	}).Methods("GET")

	router.HandleFunc("/api/host/{hname}/zone/{zname}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		res := stubZone
		stubZone.Host = vars["hname"]
		stubZone.Name = vars["zname"]

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

	router.Use(LogWrap)
	router.Use(RecoverWrap)

	return http.ListenAndServe(":3000", router)
}

func main() {
	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
