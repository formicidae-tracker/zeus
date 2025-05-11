package main

import (
	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/libarke/src-go/arke"
)

func MakeZeusControlPoint(ID arke.NodeID, temperature, humidity int16) socketcan.CanFrame {
	res := socketcan.CanFrame{
		ID:       arke.MakeCANIDT(arke.StandardMessage, arke.ZeusControlPointMessage, ID),
		Extended: false,
		Data:     make([]byte, 8),
	}
	m := arke.ZeusControlPoint{
		Humidity:    humidity,
		Temperature: temperature,
	}

	dlc, _ := m.Marshal(res.Data)
	res.Dlc = byte(dlc)
	return res
}
