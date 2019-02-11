package main

import (
	"fmt"

	"git.tuleu.science/fort/libarke/src-go/arke"
	socketcan "github.com/atuleu/golang-socketcan"
)

type Device struct {
	Class arke.NodeClass
	intf  *socketcan.RawInterface
	ID    arke.NodeID
}

type DeviceDefinition struct {
	Class        string
	CANInterface string `yaml:"can-interface"`
	ID           uint
}

func (d *Device) SendMessage(m arke.SendableMessage) error {
	return arke.SendMessage(d.intf, m, false, d.ID)
}

func (d *Device) SendResetRequest() error {
	//TODO replace with a manager
	return arke.SendResetRequest(d.intf, d.Class)
}

var nameToNodeClass = map[string]arke.NodeClass{
	"Zeus":    arke.ZeusClass,
	"Celaeno": arke.CelaenoClass,
	"Helios":  arke.HeliosClass,
}

func NameToArkeNodeClass(s string) (arke.NodeClass, error) {
	if c, ok := nameToNodeClass[s]; ok == true {
		return c, nil
	}
	return arke.NodeClass(0), fmt.Errorf("Unknown node class '%s'", s)
}
