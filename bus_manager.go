package main

import (
	"log"
	"sync"
	"syscall"
	"time"

	"git.tuleu.science/fort/libarke/src-go/arke"
	"github.com/atuleu/golang-socketcan"
)

type BusManager interface {
	Listen()
	AssignCapability(capability, arke.NodeID)
	Close() error
}

type deviceDefinition struct {
	Class arke.NodeClass
	ID    arke.NodeID
}

type messageDefinition struct {
	ID        arke.NodeID
	MessageID arke.MessageClass
}

type busManager struct {
	name         string
	intf         socketcan.RawInterface
	capabilities []capability
	devices      map[deviceDefinition]*Device
	callbacks    map[messageDefinition][]callback
	alarms       chan<- Alarm
}

func (b *busManager) Listen() {
	allClasses := map[arke.NodeClass]bool{}
	receivedHeartbeat := map[deviceDefinition]bool{}

	for d, _ := range b.devices {
		allClasses[d.Class] = true
		receivedHeartbeat[d] = false
	}

	for c, _ := range allClasses {
		arke.SendHeartBeatRequest(b.intf, c, HeartBeatPeriod)
	}

	type messageWithID struct {
		ID      arke.NodeID
		Message arke.ReceivableMessage
	}

	frames := make(chan messageWithID, 10)

	go func(frames chan<- messageWithID, intf socketcan.RawInterface) {
		for {
			f, err := intf.Receive()
			if err != nil {
				if errno, ok := err.(syscall.Errno); ok == true {
					if errno == syscall.EBADF || errno == syscall.ENETDOWN || errno == syscall.ENODEV {
						close(frames)
						log.Printf("Closed CAN Interface '%s': %s", b.name, err)
						return
					}
				}
				log.Printf("Could not receive CAN frame on '%s': %s", b.name, err)
			}
			m, ID, err := arke.ParseMessage(&f)
			if err != nil {
				log.Printf("Could not parse CAN Frame on '%s': %s", b.name, err)
			}
			frames <- messageWithID{
				ID:      ID,
				Message: m,
			}
		}
	}(frames, b.intf)

	heartbeatTimeout := time.NewTicker(3 * HeartBeatPeriod)
	defer heartbeatTimeout.Stop()
	wg := sync.WaitGroup{}
	for {
		select {
		case m, ok := <-frames:
			if ok == false {
				wg.Wait()
				return
			}
			if m.Message.MessageClassID() == arke.HeartBeatMessage {
				def := deviceDefinition{ID: m.ID, Class: m.Message.(*arke.HeartBeatData).Class}
				receivedHeartbeat[def] = true
			} else {
				mDef := messageDefinition{MessageID: m.Message.MessageClassID(), ID: m.ID}
				if callbacks, ok := b.callbacks[mDef]; ok == true {
					wg.Add(1)
					go func(m arke.ReceivableMessage, alarms chan<- Alarm) {
						for _, callback := range callbacks {
							callback(alarms, m)
						}
						wg.Done()
					}(m.Message, b.alarms)
				}
			}
		case <-heartbeatTimeout.C:
			for d, ok := range receivedHeartbeat {
				if ok == false {
					b.alarms <- NewMissingDeviceAlarm(b.name, d.Class, d.ID)
				}
				receivedHeartbeat[d] = false
			}
		}
	}
}

func (b *busManager) AssignCapability(c capability, ID arke.NodeID) {
	b.capabilities = append(b.capabilities, c)
	for _, class := range c.Requirements() {
		def := deviceDefinition{
			Class: class,
			ID:    ID,
		}
		if _, ok := b.devices[def]; ok == false {
			b.devices[def] = &Device{
				intf:  b.intf,
				Class: class,
				ID:    ID,
			}
		}
	}

	for messageClass, callback := range c.Callbacks() {
		mDef := messageDefinition{
			MessageID: messageClass,
			ID:        ID,
		}
		b.callbacks[mDef] = append(b.callbacks[mDef], callback)
	}

}

func (b *busManager) Close() error {
	return b.intf.Close()
}

func NewBusManager(interfaceName string, alarms chan<- Alarm) (BusManager, error) {
	intf, err := socketcan.NewRawInterface(interfaceName)
	if err != nil {
		return nil, err
	}
	return &busManager{
		name:      interfaceName,
		intf:      intf,
		callbacks: make(map[messageDefinition][]callback),
		devices:   make(map[deviceDefinition]*Device),
		alarms:    alarms,
	}, nil
}
