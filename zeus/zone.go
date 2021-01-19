package main

import (
	"time"

	"github.com/adrg/xdg"
	"github.com/formicidae-tracker/zeus"
)

func climateReportFilePath() (string, error) {
	return xdg.DataFile("fort-experiments/climate/climate." + time.Now().Format("2006-01-02T15:04:05.000") + ".txt")
}

func ComputeZoneRequirements(z *zeus.ZoneClimate, reporters []ClimateReporter) ([]capability, error) {
	res := []capability{}

	needClimateReport := false
	if zeus.IsUndefined(z.MinimalTemperature) == false || zeus.IsUndefined(z.MaximalTemperature) == false {
		needClimateReport = true
	}
	if zeus.IsUndefined(z.MinimalHumidity) == false || zeus.IsUndefined(z.MaximalHumidity) == false {
		needClimateReport = true
	}

	reportFileName, err := climateReportFilePath()
	if err != nil {
		return nil, err
	}
	fn, _, err := NewFileClimateReporter(reportFileName)
	if err != nil {
		return res, err
	}
	reporters = append(reporters, fn)
	go fn.Report()

	if needClimateReport == true || len(reporters) != 0 {

		chans := []chan<- zeus.ClimateReport{}
		for _, n := range reporters {
			chans = append(chans, n.ReportChannel())
		}

		res = append(res, NewClimateRecordableCapability(z.MinimalTemperature,
			z.MaximalTemperature,
			z.MinimalHumidity,
			z.MaximalHumidity,
			chans))

	}
	controlLight := false
	controlTemperature := false
	controlHumidity := false
	controlWind := false
	for _, s := range z.States {
		if zeus.IsUndefined(s.Humidity) == false {
			controlHumidity = true
		}
		if zeus.IsUndefined(s.Temperature) == false {
			controlTemperature = true
		}
		if zeus.IsUndefined(s.Wind) == false {
			controlWind = true
		}
		if zeus.IsUndefined(s.VisibleLight) == false || zeus.IsUndefined(s.UVLight) == false {
			controlLight = true
		}
	}

	if controlTemperature == true || controlWind == true {
		res = append(res, NewClimateControllable(controlHumidity))
	}

	if controlLight == true {
		res = append(res, NewLightControllable())
	}

	return res, nil
}
