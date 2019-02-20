package main

import (
	"testing"

	"git.tuleu.science/fort/libarke/src-go/arke"
	. "gopkg.in/check.v1"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/golang/mock/gomock"
)

type DeviceSuite struct {
	ctrl   *gomock.Controller
	intf   *MockRawInterface
	device Device
}

var _ = Suite(&DeviceSuite{})

func (s *DeviceSuite) SetUpMock(t *testing.T) {
	s.ctrl = gomock.NewController(t)
	s.intf = NewMockRawInterface(s.ctrl)
	s.device = Device{
		Class: arke.ZeusClass,
		ID:    1,
		intf:  s.intf,
	}
}

func TestSendMessage(t *testing.T) {
	s := DeviceSuite{}
	s.SetUpMock(t)
	defer s.ctrl.Finish()

	gomock.InOrder(
		s.intf.EXPECT().Send(socketcan.CanFrame{
			ID:   (uint32(arke.StandardMessage)<<9 | uint32(s.device.Class)<<3 | uint32(s.device.ID)),
			Dlc:  5,
			Data: []byte{0, 0, 0, 0, 0, 0, 0, 0},
		}),
		s.intf.EXPECT().Send(socketcan.CanFrame{
			ID:   (uint32(s.device.Class) << 3),
			Dlc:  1,
			Data: []byte{byte(s.device.ID), 0, 0, 0, 0, 0, 0, 0},
		}))

	s.device.SendMessage(&arke.ZeusSetPoint{Temperature: -40.0})
	s.device.SendResetRequest()
}

func (s *DeviceSuite) TestNameMapping(c *C) {
	testdata := []struct {
		Name  string
		Class arke.NodeClass
	}{
		{Name: "Zeus", Class: arke.ZeusClass},
		{Name: "zeus", Class: arke.ZeusClass},
		{Name: "zEUs", Class: arke.ZeusClass},
		{Name: "Celaeno", Class: arke.CelaenoClass},
		{Name: "celaeno", Class: arke.CelaenoClass},
		{Name: "Helios", Class: arke.HeliosClass},
		{Name: "helios", Class: arke.HeliosClass},
	}

	for _, d := range testdata {
		res, err := NameToArkeNodeClass(d.Name)
		if c.Check(err, IsNil) == false {
			continue
		}
		c.Check(res, Equals, d.Class)
	}

	_, err := NameToArkeNodeClass("hades")
	c.Check(err, ErrorMatches, `Unknown node class 'hades'`)
}
