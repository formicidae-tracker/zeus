package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"git.tuleu.science/fort/dieu"
	"git.tuleu.science/fort/libarke/src-go/arke"
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

	http.HandleFunc("/api/zone/", func(w http.ResponseWriter, r *http.Request) {
		zoneWHost := strings.TrimPrefix(r.URL.EscapedPath(), "/api/zone/")
		log.Printf(r.URL.EscapedPath())
		zone := strings.TrimPrefix(path.Ext(zoneWHost), ".")
		host := strings.TrimSuffix(zoneWHost, "."+zone)

		if len(zone) == 0 || len(host) == 0 {
			http.NotFound(w, r)
			return
		}
		res := RegisteredZone{
			Host:        host,
			Name:        zone,
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
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	http.Handle("/", http.FileServer(http.Dir("./webapp/dist/webapp")))
	return http.ListenAndServe(":3000", nil)
}

func main() {
	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
