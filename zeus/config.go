package main

import (
	"fmt"
	"io/ioutil"
	"regexp"

	flags "github.com/jessevdk/go-flags"
	yaml "gopkg.in/yaml.v2"
)

type ZoneDefinition struct {
	CANInterface string `yaml:"can-interface"`
	DevicesID    uint   `yaml:"devices-id"`
}

func (d ZoneDefinition) ID() string {
	return fmt.Sprintf("%s/%d", d.CANInterface, d.DevicesID)
}

type Config struct {
	Olympus    string                    `yaml:"olympus"`
	Interfaces map[string]string         `yaml:"interfaces"`
	Zones      map[string]ZoneDefinition `yaml:"zones"`
}

const DEFAULT_CONFIG_PATH = "/etc/default/zeus.yml"

func OpenConfigFromArg(option flags.Filename) (*Config, error) {
	configPath := DEFAULT_CONFIG_PATH
	if len(option) > 0 {
		configPath = string(option)
	}
	return OpenConfig(configPath)
}

func OpenConfig(filename string) (*Config, error) {
	c := &Config{}
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c Config) checkZones() error {
	mapping := map[string]string{}
	for name, definition := range c.Zones {
		if _, ok := c.Interfaces[definition.CANInterface]; ok == false {
			return fmt.Errorf("Invalid zone definition '%s': undefined CAN interface '%s'", name, definition.CANInterface)
		}
		if definition.DevicesID > 7 || definition.DevicesID == 0 {
			return fmt.Errorf("Invalid zone definition '%s': invalid devices-id %d ( should be in [1,7])", name, definition.DevicesID)
		}

		def := definition.ID()

		if oName, ok := mapping[def]; ok == true {
			return fmt.Errorf("Invalid zone definition '%s': devices ID %d on interface '%s' are used by zone '%s'", name, definition.DevicesID, definition.CANInterface, oName)
		}
		mapping[def] = name
	}
	return nil
}

func (c Config) checkInterfaces() error {
	mapping := map[string]string{}
	rx := regexp.MustCompile(`slcan[0-9]+`)
	for ifname, devname := range c.Interfaces {
		if rx.MatchString(ifname) == false {
			return fmt.Errorf("Invalid interface definition '%s': invalid interface name", ifname)
		}

		if oName, ok := mapping[devname]; ok == true {
			return fmt.Errorf("Invalid interface definition '%s': device '%s' is already used by interface %s", ifname, devname, oName)
		}
		mapping[devname] = ifname
	}
	return nil
}

func (c Config) Check() error {
	if err := c.checkInterfaces(); err != nil {
		return err
	}
	return c.checkZones()
}
