package main

import "git.tuleu.science/fort/dieu"

func ComputeZoneRequirements(z *dieu.Zone) ([]capability, error) {
	res := []capability{}

	notifiers := []ClimateReporter{}
	needClimateReport := false
	if dieu.IsUndefined(z.MinimalTemperature) == false || dieu.IsUndefined(z.MaximalTemperature) == false {
		needClimateReport = true
	}
	if dieu.IsUndefined(z.MinimalHumidity) == false || dieu.IsUndefined(z.MaximalHumidity) == false {
		needClimateReport = true
	}
	if len(z.ClimateReportFile) != 0 {
		fn, _, err := NewFileClimateReporter(z.ClimateReportFile)
		if err != nil {
			return res, err
		}
		notifiers = append(notifiers, fn)
	}

	if needClimateReport == true || len(notifiers) != 0 {

		chans := []chan<- dieu.ClimateReport{}
		for _, n := range notifiers {
			chans = append(chans, n.C())
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
		if dieu.IsUndefined(s.Humidity) == false {
			controlHumidity = true
		}
		if dieu.IsUndefined(s.Temperature) == false {
			controlTemperature = true
		}
		if dieu.IsUndefined(s.Wind) == false {
			controlWind = true
		}
		if dieu.IsUndefined(s.VisibleLight) == false || dieu.IsUndefined(s.UVLight) == false {
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
