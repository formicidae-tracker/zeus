package zeus

import (
	"errors"

	. "gopkg.in/check.v1"
)

type ZeusRPCSuite struct{}

var _ = Suite(&ZeusRPCSuite{})

func (s *ZeusRPCSuite) TestZoneIdentifier(c *C) {
	testdata := []struct {
		Host, Name, Expected string
	}{
		{"hello", "world", "hello/zone/world"},
	}

	for _, d := range testdata {
		c.Check(ZoneIdentifier(d.Host, d.Name), Equals, d.Expected)
		c.Check(ZoneRegistration{Host: d.Host, Name: d.Name}.Fullname(), Equals, d.Expected)
		c.Check(ZoneUnregistration{Host: d.Host, Name: d.Name}.Fullname(), Equals, d.Expected)
	}
}

func (s *ZeusRPCSuite) TestHermesError(c *C) {
	c.Check(ZeusError("").ToError(), Equals, nil)
	err := errors.New("some error")
	c.Check(ZeusError(err.Error()).ToError(), ErrorMatches, err.Error())
}
