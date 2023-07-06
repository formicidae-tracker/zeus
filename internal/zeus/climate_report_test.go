package zeus

import (
	"math"

	. "gopkg.in/check.v1"
)

type ClimateReportSuite struct{}

var _ = Suite(&ClimateReportSuite{})

func (s *ClimateReportSuite) TestGoodClimateReportContainsNoNaN(c *C) {
	testdata := []struct {
		Report ClimateReport
		Error  string
	}{
		{ClimateReport{Humidity: Humidity(math.NaN())}, "humidity NaN"},
		{ClimateReport{Temperatures: []Temperature{Temperature(math.NaN()), 0, 0, 0}}, "temperature 0 NaN"},
		{ClimateReport{Temperatures: []Temperature{0, Temperature(math.NaN()), 0, 0}}, "temperature 1 NaN"},
		{ClimateReport{Temperatures: []Temperature{0, 0, Temperature(math.NaN()), 0}}, "temperature 2 NaN"},
		{ClimateReport{Temperatures: []Temperature{0, 0, 0, Temperature(math.NaN())}}, "temperature 3 NaN"},
	}

	for _, d := range testdata {
		c.Check(d.Report.Check(), ErrorMatches, d.Error)
	}

	c.Check((ClimateReport{}).Check(), ErrorMatches, "no temperature")
	c.Check((ClimateReport{Temperatures: []Temperature{0}}).Check(), IsNil)
}
