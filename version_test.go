package zeus

import (
	"github.com/blang/semver"
	. "gopkg.in/check.v1"
)

type VersionSuite struct {
}

var _ = Suite(&VersionSuite{})

func (s *VersionSuite) TestParsing(c *C) {
	testdata := []struct {
		Input    string
		Expected semver.Version
	}{
		{"v0.2.0", semver.Version{Major: 0, Minor: 2, Patch: 0}},
		{"v0.2.0-15-g123559f",
			semver.Version{
				Major: 0,
				Minor: 2,
				Patch: 0,
				Pre: []semver.PRVersion{
					{VersionStr: "15-g123559f"},
				},
			},
		},
		{"v1.2.4", semver.Version{Major: 1, Minor: 2, Patch: 4}},
	}

	for _, d := range testdata {
		r, err := semver.ParseTolerant(d.Input)

		c.Check(err, IsNil)
		c.Check(r.Equals(d.Expected), Equals, true, Commentf("Input:%s Result:%s Expected:%s", d.Input, r, d.Expected))
	}

}

func (s *VersionSuite) TestCompatibility(c *C) {
	testdata := []struct {
		A, B       string
		Error      string
		Compatible bool
	}{
		{"development", "v56.34.2", "", true},
		{"v0.34.2", "development", "", true},
		{"v0.34.2", "v0.35.0-45-g32456785", "", false},
		{"v0.34.2", "v0.34.0-45-g32456785", "", true},
		{"v1.2.3-3-gaeb345bd", "v1.4.0", "", true},
		{"v1.2-3", "v1.4.0", `Invalid version 'v1.2-3':.*`, false},
		{"v1.2.0", "v1.4-0", `Invalid version 'v1.4-0':.*`, false},
	}

	for _, d := range testdata {
		comment := Commentf("Testing %+v", d)
		r, err := VersionAreCompatible(d.A, d.B)
		if len(d.Error) == 0 {
			c.Check(err, IsNil, comment)
			c.Check(r, Equals, d.Compatible, comment)
		} else {
			c.Check(r, Equals, false, comment)
			c.Check(err, ErrorMatches, d.Error, comment)
		}
	}

}
