package zeus

import (
	. "gopkg.in/check.v1"
)

type OlympusRPCSuite struct{}

var _ = Suite(&OlympusRPCSuite{})

func (s *OlympusRPCSuite) TestZoneIdentifier(c *C) {
	testdata := []struct {
		Host, Name, Expected string
	}{
		{"hello", "world", "hello/zone/world"},
	}

	for _, d := range testdata {
		c.Check(ZoneIdentifier(d.Host, d.Name), Equals, d.Expected)
		c.Check(ZoneRegistration{Host: d.Host, Name: d.Name}.ZoneIdentifier(), Equals, d.Expected)
		c.Check(ZoneUnregistration{Host: d.Host, Name: d.Name}.ZoneIdentifer(), Equals, d.Expected)
	}
}
