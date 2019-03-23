package main

import (
	"fmt"
	"strings"

	"github.com/formicidae-tracker/dieu"
	"github.com/formicidae-tracker/libarke/src-go/arke"
	socketcan "github.com/atuleu/golang-socketcan"
)

type Device struct {
	Class arke.NodeClass
	intf  socketcan.RawInterface
	ID    arke.NodeID
}

func (d *Device) SendMessage(m arke.SendableMessage) error {
	return arke.SendMessage(d.intf, m, false, d.ID)
}

func (d *Device) SendResetRequest() error {
	return arke.SendResetRequest(d.intf, d.Class, d.ID)
}

func (d *Device) SendHeartbeatRequest() error {
	return arke.SendHeartBeatRequest(d.intf, d.Class, dieu.HeartBeatPeriod)
}

var nameToNodeClass = map[string]arke.NodeClass{
	"zeus":    arke.ZeusClass,
	"celaeno": arke.CelaenoClass,
	"helios":  arke.HeliosClass,
}

var nodeClassToNode = map[arke.NodeClass]string{
	arke.ZeusClass:    "Zeus",
	arke.CelaenoClass: "Celaeno",
	arke.HeliosClass:  "Helios",
}

func NameToArkeNodeClass(s string) (arke.NodeClass, error) {
	if c, ok := nameToNodeClass[strings.ToLower(s)]; ok == true {
		return c, nil
	}
	return arke.NodeClass(0), fmt.Errorf("Unknown node class '%s'", s)
}

func Name(c arke.NodeClass) string {
	if n, ok := nodeClassToNode[c]; ok == true {
		return n
	}
	return "<unknown>"
}
