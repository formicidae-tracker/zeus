package dieu

import "time"

type ClimateReport struct {
	Temperatures [4]Temperature
	Humidity     Humidity
	Time         time.Time
}

type NamedClimateReport struct {
	ClimateReport
	ZoneIdentifier string
}
