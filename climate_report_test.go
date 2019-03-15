package dieu

import (
	"math"

	. "gopkg.in/check.v1"
)

type ClimateReportSuite struct{}

var _ = Suite(&ClimateReportSuite{})

func (s *ClimateReportSuite) TestGoodClimateReportContainsNoNaN(c *C) {
	testdata := []ClimateReport{
		{Humidity: Humidity(math.NaN())},
		{Temperatures: [4]Temperature{Temperature(math.NaN()), 0, 0, 0}},
		{Temperatures: [4]Temperature{0, Temperature(math.NaN()), 0, 0}},
		{Temperatures: [4]Temperature{0, 0, Temperature(math.NaN()), 0}},
		{Temperatures: [4]Temperature{0, 0, 0, Temperature(math.NaN())}},
	}

	for _, d := range testdata {
		c.Check(d.Good(), Equals, false)
	}

	c.Check(ClimateReport{}.Good(), Equals, true)
}
