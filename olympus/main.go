package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"git.tuleu.science/fort/dieu"
	"git.tuleu.science/fort/libarke/src-go/arke"
	"github.com/gorilla/mux"
)

type RegisteredAlarm struct {
	Reason     string
	On         bool
	Level      int
	LastChange *time.Time
	Triggers   int
}

type RegisteredZone struct {
	Host        string
	Name        string
	Temperature float64
	Humidity    float64
	Alarms      []RegisteredAlarm
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

		data, err := json.Marshal(&res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

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

		data, err := json.Marshal(&res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	router.HandleFunc("/api/host/{hname}/zone/{zname}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		res := RegisteredZone{
			Host:        vars["hname"],
			Name:        vars["zname"],
			Temperature: 21.2,
			Humidity:    62,
			Alarms:      []RegisteredAlarm{},
		}
		alarms := []dieu.Alarm{
			dieu.WaterLevelWarning,
			dieu.WaterLevelCritical,
			dieu.TemperatureOutOfBound,
			dieu.HumidityOutOfBound,
			dieu.TemperatureUnreachable,
			dieu.HumidityUnreachable,
			dieu.NewMissingDeviceAlarm("slcan0", arke.ZeusClass, 1),
			dieu.NewMissingDeviceAlarm("slcan0", arke.CelaenoClass, 1),
			dieu.NewMissingDeviceAlarm("slcan0", arke.HeliosClass, 1),
		}
		for _, a := range alarms {
			aa := RegisteredAlarm{
				Reason:   a.Reason(),
				On:       false,
				Triggers: 0,
			}
			if a.Priority() == dieu.Warning {
				aa.Level = 1
			} else {
				aa.Level = 2
			}

			res.Alarms = append(res.Alarms, aa)
		}

		data, err := json.Marshal(&res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)

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

	return http.ListenAndServe(":3000", router)
}

func main() {
	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
