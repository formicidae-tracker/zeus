package main

import "git.tuleu.science/fort/dieu"

func ComputeZoneRequirements(z *dieu.Zone) ([]capability, error) {
	res := []capability{}
	if dieu.IsUndefined(z.MinimalTemperature) == false || dieu.IsUndefined(z.MaximalTemperature) == false || dieu.IsUndefined(z.MinimalHumidity) == false || dieu.IsUndefined(z.MaximalHumidity) == false || len(z.ClimateReportFile) != 0 {
		if toAdd, err := NewClimateRecordableCapability(z.MinimalTemperature,
			z.MaximalTemperature,
			z.MinimalHumidity,
			z.MaximalHumidity,
			z.ClimateReportFile); err != nil {
			return res, err
		} else {
			res = append(res, toAdd)
		}
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
