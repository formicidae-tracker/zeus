package main

import (
	"fmt"
	"log"
	"os"
	"time"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/formicidae-tracker/zeus"
)

type DeviceDefinition struct {
	Class arke.NodeClass
	ID    arke.NodeID
}

type PresenceMonitorer interface {
	Monitor([]DeviceDefinition, chan<- TimedAlarm)
	Ping(class arke.NodeClass, ID arke.NodeID)
	Close() error
}

type presenceMonitorer struct {
	HeartBeatPeriod time.Duration

	quit, done chan struct{}
	pings      chan DeviceDefinition
	logger     *log.Logger

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

func (m *presenceMonitorer) Monitor(devices []DeviceDefinition, alarms chan<- TimedAlarm) {

	if m.quit != nil {
		return
	}
	m.quit = make(chan struct{})
	m.done = make(chan struct{})
	defer close(m.done)

	received := make(map[DeviceDefinition]bool)
	for _, d := range devices {
		received[d] = false
		arke.SendHeartBeatRequest(m.intf, d.Class, m.HeartBeatPeriod)
	}

	timeout := time.NewTicker(3 * m.HeartBeatPeriod)

	defer timeout.Stop()
	for {
		select {
		case <-m.quit:
			return
		case def := <-m.pings:
			if _, ok := received[def]; ok == false {
				m.logger.Printf("unmonitored device %+v", def)
				continue
			}
			received[def] = true
		case t := <-timeout.C:
			deviceRequest := make(map[arke.NodeClass]bool)
			for d, ok := range received {
				if ok == true {
					received[d] = false
					continue
				}
				alarms <- TimedAlarm{Alarm: zeus.NewMissingDeviceAlarm(m.ifname, d.Class, d.ID), Time: t}
				deviceRequest[d.Class] = true
			}
			for c, _ := range deviceRequest {
				arke.SendHeartBeatRequest(m.intf, c, m.HeartBeatPeriod)
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
		logger:          log.New(os.Stderr, "[monitor/"+ifname+"] ", 0),
		ifname:          ifname,
		intf:            intf,
	}
}
