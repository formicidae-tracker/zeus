package dieu

import (
	"math"
	"time"
)

type ClimateReport struct {
	Temperatures [4]Temperature
	Humidity     Humidity
	Time         time.Time
}

func (r ClimateReport) Good() bool {
	if math.IsNaN(r.Humidity.Value()) == true {
		return false
	}
	for i := 0; i < 4; i++ {
		if math.IsNaN(r.Temperatures[i].Value()) == true {
			return false
		}
	}
	return true
}

type NamedClimateReport struct {
	ClimateReport
	ZoneIdentifier string
}
