package main

type Zone struct {
	Devices            []DeviceDefinition
	ClimateReportFile  string
	MinimalTemperature Temperature
	MaximalTemperature Temperature
	MinimalHumidity    Humidity
	MaximalHumidity    Humidity
	States             []State
	Transitions        []Transition
}

func (z *Zone) ComputeRequirements() []capability {
	res := []capability{}
	if IsUndefined(z.MinimalTemperature) == false || IsUndefined(z.MaximalTemperature) == false || IsUndefined(z.MinimalHumidity) == false || IsUndefined(z.MaximalHumidity) == false || len(z.ClimateReportFile) != 0 {
		res = append(res, &ClimateRecordable{})
	}
	controlLight := false
	controlTemperature := false
	controlHumidity := false
	controlWind := false
	for _, s := range z.States {
		if IsUndefined(s.Humidity) == false {
			controlHumidity = true
		}
		if IsUndefined(s.Temperature) == false {
			controlTemperature = true
		}
		if IsUndefined(s.Wind) == false {
			controlWind = true
		}
		if IsUndefined(s.VisibleLight) == false || IsUndefined(s.UVLight) == false {
			controlLight = true
		}
	}

	if controlTemperature == true || controlWind == true {
		res = append(res, NewClimateControllable(controlHumidity))
	}

	if controlLight == true {
		res = append(res, NewLightControllable())
	}

	return res
}
