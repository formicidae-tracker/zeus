package main

import (
	"fmt"
	"path"
	"time"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/formicidae-tracker/olympus/pkg/tm"
	"github.com/formicidae-tracker/zeus/internal/zeus"
	"github.com/sirupsen/logrus"
)

type DeviceDefinition struct {
	Class arke.NodeClass
	ID    arke.NodeID
}

type PresenceMonitorer interface {
	Monitor([]DeviceDefinition, chan<- zeus.Alarm, chan<- struct{})
	Ping(class arke.NodeClass, ID arke.NodeID)
	Close() error
}

type presenceMonitorer struct {
	HeartBeatPeriod time.Duration

	quit, done chan struct{}
	pings      chan DeviceDefinition
	logger     *logrus.Entry

	ifname string
	intf   socketcan.RawInterface
}

func (m *presenceMonitorer) Close() error {
	if m.quit == nil {
		return fmt.Errorf("Already closed")
	}
	close(m.quit)
	<-m.done
	close(m.pings)
	m.quit = nil
	m.done = nil
	return nil
}

func (m *presenceMonitorer) Monitor(devices []DeviceDefinition, alarms chan<- zeus.Alarm, ready chan<- struct{}) {

	if m.quit != nil {
		return
	}
	m.quit = make(chan struct{})
	m.done = make(chan struct{})
	defer close(m.done)

	received := make(map[DeviceDefinition]bool)
	for _, d := range devices {
		received[d] = false
		m.intf.Send(arke.MakeHeartBeatRequest(d.Class, m.HeartBeatPeriod))
	}

	timeout := time.NewTicker(3 * m.HeartBeatPeriod)

	defer timeout.Stop()
	close(ready)
	for {
		select {
		case <-m.quit:
			return
		case def := <-m.pings:
			if _, ok := received[def]; ok == false {
				m.logger.WithField("device", def).Warn("unmonitored device")
				continue
			}
			received[def] = true
		case <-timeout.C:
			deviceRequest := make(map[arke.NodeClass]bool)
			for d, ok := range received {
				if ok == true {
					received[d] = false
					continue
				}
				alarms <- zeus.NewMissingDeviceAlarm(m.ifname, d.Class, d.ID)
				deviceRequest[d.Class] = true
			}
			for c, _ := range deviceRequest {
				m.intf.Send(arke.MakeHeartBeatRequest(c, m.HeartBeatPeriod))
			}
		}
	}
}

func (m *presenceMonitorer) Ping(class arke.NodeClass, ID arke.NodeID) {
	m.pings <- DeviceDefinition{Class: class, ID: ID}
}

func NewPresenceMonitorer(ifname string, intf socketcan.RawInterface) PresenceMonitorer {
	return &presenceMonitorer{
		HeartBeatPeriod: zeus.HeartBeatPeriod,
		pings:           make(chan DeviceDefinition, 5),
		logger:          tm.NewLogger(path.Join("monitor", ifname)),
		ifname:          ifname,
		intf:            intf,
	}
}
