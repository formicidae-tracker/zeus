package zeus

import (
	"fmt"
	"math"
	"time"
)

type ClimateReport struct {
	Temperatures []Temperature
	Humidity     Humidity
	Time         time.Time
}

func (r ClimateReport) Check() error {
	if math.IsNaN(r.Humidity.Value()) == true {
		return fmt.Errorf("humidity NaN")
	}

	if len(r.Temperatures) == 0 {
		return fmt.Errorf("no temperature")
	}

	for i, t := range r.Temperatures {
		if math.IsNaN(t.Value()) == true {
			return fmt.Errorf("temperature %d NaN", i)
		}
	}
	return nil
}

type NamedClimateReport struct {
	ClimateReport
	ZoneIdentifier string
}
