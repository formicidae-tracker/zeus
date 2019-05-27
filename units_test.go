package zeus

import (
	. "gopkg.in/check.v1"
)

type UnitsSuite struct{}

var _ = Suite(&UnitsSuite{})

func (s *UnitsSuite) TestBoundaries(c *C) {
	var t Temperature = UndefinedTemperature
	var h Humidity
	var w Wind
	var l Light
	c.Check(t.MaxValue(), Equals, 40.0)
	c.Check(t.MinValue(), Equals, 15.0)
	c.Check(IsUndefined(t), Equals, true)

	c.Check(h.MaxValue(), Equals, 85.0)
	c.Check(h.MinValue(), Equals, 10.0)
	c.Check(IsUndefined(h), Equals, false)

	c.Check(w.MaxValue(), Equals, 100.0)
	c.Check(w.MinValue(), Equals, 0.0)
	c.Check(IsUndefined(w), Equals, false)

	c.Check(l.MaxValue(), Equals, 100.0)
	c.Check(l.MinValue(), Equals, 0.0)
	c.Check(IsUndefined(l), Equals, false)

	c.Check(Clamp(t), Equals, t.MinValue())

}
