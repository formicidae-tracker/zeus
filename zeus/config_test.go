package main

import (
	"io/ioutil"
	"os"

	. "gopkg.in/check.v1"
)

type ConfigSuite struct {
}

var _ = Suite(&ConfigSuite{})

var complexConfig = &Config{
	Interfaces: map[string]string{
		"slcan0": "/dev/ttyS0",
		"slcan1": "/dev/ttyS1",
	},
	Zones: map[string]ZoneDefinition{
		"box": ZoneDefinition{
			CANInterface: "slcan0",
			DevicesID:    1,
		},
		"tunnel": ZoneDefinition{
			CANInterface: "slcan0",
			DevicesID:    2,
		},
		"nest": ZoneDefinition{
			CANInterface: "slcan1",
			DevicesID:    1,
		},
	},
}

func (s *ConfigSuite) TestLoad(c *C) {
	tmpfile, err := ioutil.TempFile("", "zeus-tests")
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name())

	content := `---
interfaces:
  slcan0: /dev/ttyS0
  slcan1: /dev/ttyS1
zones:
  box:
    can-interface: slcan0
    devices-id: 1
  tunnel:
    can-interface: slcan0
    devices-id: 2
  nest:
    can-interface: slcan1
    devices-id: 1
`
	_, err = tmpfile.Write([]byte(content))
	c.Assert(err, IsNil)

	config, err := OpenConfig(tmpfile.Name())
	c.Assert(err, IsNil)
	c.Check(config, DeepEquals, complexConfig)
	_, err = OpenConfig("does-not-exist")
	c.Check(err, Not(IsNil))
	_, err = tmpfile.Write([]byte(`asfjjp:fdflj:dfjdskf
fsdj: sdf
`))
	c.Assert(err, IsNil)

	_, err = OpenConfig(tmpfile.Name())
	c.Check(err, Not(IsNil))
}

func (s *ConfigSuite) TestErrorChecking(c *C) {
	testdata := map[*Config]string{
		&Config{}:     "",
		complexConfig: "",
		&Config{
			Interfaces: map[string]string{
				"dlcan0": "/dev/ttyS0",
			},
		}: "Invalid interface definition 'dlcan0': invalid interface name",
		&Config{
			Interfaces: map[string]string{
				"slcan0": "/dev/ttyS0",
				"slcan1": "/dev/ttyS0",
			},
		}: "Invalid interface definition 'slcan.*': device '/dev/ttyS0' is already used by interface slcan.*",
		&Config{
			Interfaces: map[string]string{
				"slcan0": "/dev/ttyS0",
			},
			Zones: map[string]ZoneDefinition{
				"box": ZoneDefinition{
					CANInterface: "slcan1",
					DevicesID:    1,
				},
			},
		}: "Invalid zone definition 'box': undefined CAN interface 'slcan1'",
		&Config{
			Interfaces: map[string]string{
				"slcan0": "/dev/ttyS0",
			},
			Zones: map[string]ZoneDefinition{
				"box": ZoneDefinition{
					CANInterface: "slcan0",
					DevicesID:    42,
				},
			},
		}: "Invalid zone definition 'box': invalid devices-id .*",
		&Config{
			Interfaces: map[string]string{
				"slcan0": "/dev/ttyS0",
			},
			Zones: map[string]ZoneDefinition{
				"box": ZoneDefinition{
					CANInterface: "slcan0",
					DevicesID:    0,
				},
			},
		}: "Invalid zone definition 'box': invalid devices-id .*",
		&Config{
			Interfaces: map[string]string{
				"slcan0": "/dev/ttyS0",
			},
			Zones: map[string]ZoneDefinition{
				"box": ZoneDefinition{
					CANInterface: "slcan0",
					DevicesID:    1,
				},
				"box2": ZoneDefinition{
					CANInterface: "slcan0",
					DevicesID:    1,
				},
			},
		}: "Invalid zone definition 'box.*': devices ID 1 on interface 'slcan0' are used by zone 'box.*'",
	}

	for config, expectedError := range testdata {
		err := config.Check()
		if len(expectedError) == 0 {
			c.Check(err, IsNil)
		} else {
			c.Check(err, ErrorMatches, expectedError)
		}
	}
}
