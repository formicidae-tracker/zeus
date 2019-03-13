package main

import (
	"fmt"
	"log"
	"sync"
	"syscall"
	"time"

	"git.tuleu.science/fort/dieu"
	"git.tuleu.science/fort/libarke/src-go/arke"
	socketcan "github.com/atuleu/golang-socketcan"
)

type BusManager interface {
	Listen()
	AssignCapabilitiesForID(arke.NodeID, []capability, chan<- dieu.Alarm) error
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
	name              string
	intf              socketcan.RawInterface
	capabilities      []capability
	alarms            map[arke.NodeID]chan<- dieu.Alarm
	devices           map[deviceDefinition]*Device
	callbacks         map[messageDefinition][]callback
	callbackWaitGroup sync.WaitGroup
}

func (b *busManager) receiveAndStampMessage(frames chan<- *StampedMessage) {
	for {
		f, err := b.intf.Receive()
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
		t := time.Now()
		m, ID, err := arke.ParseMessage(&f)
		if err != nil {
			log.Printf("Could not parse CAN Frame on '%s': %s", b.name, err)
		}
		frames <- &StampedMessage{
			M:  m,
			ID: ID,
			T:  t,
		}
	}
}

func (b *busManager) Listen() {
	allClasses := map[arke.NodeClass]bool{}
	receivedHeartbeat := map[deviceDefinition]bool{}

	for d, _ := range b.devices {
		allClasses[d.Class] = true
		receivedHeartbeat[d] = false
	}

	for c, _ := range allClasses {
		arke.SendHeartBeatRequest(b.intf, c, dieu.HeartBeatPeriod)
	}

	frames := make(chan *StampedMessage, 10)

	go b.receiveAndStampMessage(frames)

	heartbeatTimeout := time.NewTicker(3 * dieu.HeartBeatPeriod)
	defer heartbeatTimeout.Stop()

	for {
		select {
		case m, ok := <-frames:
			if ok == false {
				return
			}
			if m.M.MessageClassID() == arke.HeartBeatMessage {
				def := deviceDefinition{ID: m.ID, Class: m.M.(*arke.HeartBeatData).Class}
				receivedHeartbeat[def] = true
			} else {
				mDef := messageDefinition{MessageID: m.M.MessageClassID(), ID: m.ID}
				if callbacks, ok := b.callbacks[mDef]; ok == true {
					b.callbackWaitGroup.Add(1)
					go func(m *StampedMessage, alarms chan<- dieu.Alarm) {
						for _, callback := range callbacks {
							log.Printf("callback %+v", callback)
							callback(alarms, m)
						}
						b.callbackWaitGroup.Done()
					}(m, b.alarms[m.ID])
				}
			}
		case <-heartbeatTimeout.C:
			for d, ok := range receivedHeartbeat {
				if ok == false {
					b.alarms[d.ID] <- dieu.NewMissingDeviceAlarm(b.name, d.Class, d.ID)
				}
				receivedHeartbeat[d] = false
			}
		}
	}
}

func (b *busManager) assignCapabilityUnsafe(c capability, ID arke.NodeID) {
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

	deviceMap := make(map[arke.NodeClass]*Device)
	for _, class := range c.Requirements() {
		deviceMap[class] = b.devices[deviceDefinition{
			Class: class,
			ID:    ID,
		}]
	}
	c.SetDevices(deviceMap)

	for messageClass, callback := range c.Callbacks() {
		mDef := messageDefinition{
			MessageID: messageClass,
			ID:        ID,
		}
		b.callbacks[mDef] = append(b.callbacks[mDef], callback)
	}
}

func (b *busManager) AssignCapabilitiesForID(ID arke.NodeID, capabilities []capability, alarms chan<- dieu.Alarm) error {
	if _, ok := b.alarms[ID]; ok == true {
		return fmt.Errorf("ID %d is already assigned", ID)
	}
	b.alarms[ID] = alarms
	for _, c := range capabilities {
		b.assignCapabilityUnsafe(c, ID)
	}

	return nil
}

func (b *busManager) Close() error {
	err := b.intf.Close()
	b.callbackWaitGroup.Wait()
	for _, a := range b.alarms {
		close(a)
	}
	return err
}

func NewBusManager(interfaceName string) (BusManager, error) {
	intf, err := socketcan.NewRawInterface(interfaceName)
	if err != nil {
		return nil, err
	}
	return &busManager{
		name:      interfaceName,
		intf:      intf,
		callbacks: make(map[messageDefinition][]callback),
		devices:   make(map[deviceDefinition]*Device),
		alarms:    make(map[arke.NodeID]chan<- dieu.Alarm),
	}, nil
}
