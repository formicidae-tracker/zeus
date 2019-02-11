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

func NameToArkeNodeClass(s string) (arke.NodeClass, error) {
	switch s {
	case "Zeus":
		return arke.ZeusClass, nil
	case "Celaeno":
		return arke.CelaenoClass, nil
	case "Helios":
		return arke.HeliosClass, nil
	}
	return arke.NodeClass(0), fmt.Errorf("Unknown node class '%s'", s)
}
