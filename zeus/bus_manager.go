package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/formicidae-tracker/zeus"
	"github.com/formicidae-tracker/libarke/src-go/arke"
	socketcan "github.com/atuleu/golang-socketcan"
)

type BusManager interface {
	Listen()
	AssignCapabilitiesForID(arke.NodeID, []capability, chan<- zeus.Alarm) error
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
	alarms            map[arke.NodeID]chan<- zeus.Alarm
	devices           map[deviceDefinition]*Device
	callbacks         map[messageDefinition][]callback
	callbackWaitGroup *sync.WaitGroup
	listenWaitGroup   *sync.WaitGroup
	log               *log.Logger
	heartbeat         time.Duration
}

func (b *busManager) receiveAndStampMessage(frames chan<- *StampedMessage) {
	for {
		f, err := b.intf.Receive()
		if err != nil {
			if errno, ok := err.(syscall.Errno); ok == true {
				if errno == syscall.EBADF || errno == syscall.ENETDOWN || errno == syscall.ENODEV {
					close(frames)
					b.log.Printf("Closed CAN Interface: %s", err)
					return
				}
			}
			b.log.Printf("Could not receive CAN frame on: %s", err)
		} else {
			t := time.Now()
			m, ID, err := arke.ParseMessage(&f)
			if err != nil {
				b.log.Printf("Could not parse CAN Frame on: %s", err)
			} else {
				frames <- &StampedMessage{
					M:  m,
					ID: ID,
					T:  t,
				}
			}
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
		arke.SendHeartBeatRequest(b.intf, c, b.heartbeat)
	}

	frames := make(chan *StampedMessage, 10)

	go b.receiveAndStampMessage(frames)

	heartbeatTimeout := time.NewTicker(3 * b.heartbeat)
	b.listenWaitGroup.Add(1)
	defer func() {
		heartbeatTimeout.Stop()
		b.listenWaitGroup.Done()
	}()

	b.log.Printf("started listening loop")
	for {
		select {
		case m, ok := <-frames:
			if ok == false {
				b.log.Printf("ended listening loop")
				return
			}
			switch m.M.MessageClassID() {
			case arke.HeartBeatMessage:
				def := deviceDefinition{ID: m.ID, Class: m.M.(*arke.HeartBeatData).Class}
				receivedHeartbeat[def] = true
			case arke.ErrorReportMessage:
				if alarms, ok := b.alarms[m.ID]; ok == true {
					casted := m.M.(*arke.ErrorReportData)
					alarms <- zeus.NewDeviceInternalError(b.name, casted.Class, casted.ID, casted.ErrorCode)
				}
			default:
				mDef := messageDefinition{MessageID: m.M.MessageClassID(), ID: m.ID}
				if callbacks, ok := b.callbacks[mDef]; ok == true {
					b.callbackWaitGroup.Add(1)
					go func(m *StampedMessage, alarms chan<- zeus.Alarm) {
						for _, callback := range callbacks {
							if err := callback(alarms, m); err != nil {
								b.log.Printf("callback error on %s: %s", m.M, err)
							}
						}
						b.callbackWaitGroup.Done()
					}(m, b.alarms[m.ID])
				}
			}
		case <-heartbeatTimeout.C:
			var askforHeartBeat map[arke.NodeClass]bool = nil
			for d, ok := range receivedHeartbeat {
				if ok == false {
					b.alarms[d.ID] <- zeus.NewMissingDeviceAlarm(b.name, d.Class, d.ID)
					if askforHeartBeat == nil {
						askforHeartBeat = make(map[arke.NodeClass]bool)
					}
					askforHeartBeat[d.Class] = true
				}
				receivedHeartbeat[d] = false
			}
			for c, _ := range askforHeartBeat {
				arke.SendHeartBeatRequest(b.intf, c, b.heartbeat)
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

func (b *busManager) AssignCapabilitiesForID(ID arke.NodeID, capabilities []capability, alarms chan<- zeus.Alarm) error {
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
	b.listenWaitGroup.Wait()
	b.callbackWaitGroup.Wait()
	for _, a := range b.alarms {
		close(a)
	}
	return err
}

func NewBusManager(interfaceName string, intf socketcan.RawInterface, heartbeat time.Duration) BusManager {
	logger := log.New(os.Stderr, "[CAN/"+interfaceName+"]: ", log.LstdFlags)
	return &busManager{
		name:              interfaceName,
		intf:              intf,
		callbacks:         make(map[messageDefinition][]callback),
		devices:           make(map[deviceDefinition]*Device),
		alarms:            make(map[arke.NodeID]chan<- zeus.Alarm),
		log:               logger,
		callbackWaitGroup: &sync.WaitGroup{},
		listenWaitGroup:   &sync.WaitGroup{},
		heartbeat:         heartbeat,
	}
}
