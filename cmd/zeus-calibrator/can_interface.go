package main

import (
	"log/slog"
	"time"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/libarke/src-go/arke"
)

type timedMessage struct {
	T  time.Time
	ID arke.NodeID
	M  arke.ReceivableMessage
}

type canInterface struct {
	logger *slog.Logger

	Messages chan timedMessage
	intf     socketcan.RawInterface
}

func (c *canInterface) listen() {
	defer close(c.Messages)
	for {
		f, err := c.intf.Receive()
		now := time.Now()
		if err != nil {
			c.logger.Error("could not read frame", "error", err)
			return
		}
		m, id, err := arke.ParseMessage(&f)
		if err != nil {
			c.logger.Error("could not parse frame", "error", err)
			continue
		}

		c.Messages <- timedMessage{T: now, M: m, ID: id}
	}
}

func OpenCanInterface(name string) (*canInterface, error) {
	intf, err := socketcan.NewRawInterface(name)
	if err != nil {
		return nil, err
	}

	res := &canInterface{
		Messages: make(chan timedMessage, 100),
		intf:     intf,
		logger:   slog.With("canInterface", name),
	}

	go res.listen()
	return res, nil
}

func (intf *canInterface) Close() error {
	intf.logger.Info("closing")
	return intf.Close()
}

func (intf *canInterface) Send(c socketcan.CanFrame) error {
	intf.logger.Debug("sending frame", "frame", c)
	return intf.intf.Send(c)
}

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
