package main

import (
	"log"
	"os"
	"sync"
	"syscall"
	"time"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/libarke/src-go/arke"
)

type StampedMessage struct {
	M  arke.ReceivableMessage
	T  time.Time
	ID arke.NodeID
}

type ArkeDispatcher interface {
	Dispatch()
	Register(devicesID arke.NodeID) <-chan *StampedMessage
	Close() error
}

type arkeDispatcher struct {
	mx       sync.RWMutex
	channels map[int][]chan *StampedMessage

	intf   socketcan.RawInterface
	logger *log.Logger
	done   chan struct{}
}

func (d *arkeDispatcher) closeChannels() {
	d.mx.Lock()
	defer d.mx.Unlock()

	for _, channels := range d.channels {
		for _, channel := range channels {
			close(channel)
		}
	}
	d.channels = nil
}

func (d *arkeDispatcher) nonBlockingSend(m *StampedMessage, c chan<- *StampedMessage) {
	select {
	case c <- m:
		return
	default:
		d.logger.Printf("One receiver ready for ID %d isn't ready, dropping message", m.ID)
	}
}

func (d *arkeDispatcher) dispatchMessage(m *StampedMessage) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	channels, ok := d.channels[int(m.ID)]
	if ok == false {
		return
	}
	for _, channel := range channels {
		d.nonBlockingSend(m, channel)
	}
}

func (d *arkeDispatcher) Dispatch() {
	d.done = make(chan struct{})
	defer close(d.done)
	for {
		f, err := d.intf.Receive()
		if err != nil {
			if errno, ok := err.(syscall.Errno); ok == true {
				if errno == syscall.EBADF || errno == syscall.ENETDOWN || errno == syscall.ENODEV {
					return
				}
			}
			d.logger.Printf("Could not receive CAN frame on: %s", err)
		} else {
			t := time.Now()
			m, ID, err := arke.ParseMessage(&f)
			if err != nil {
				d.logger.Printf("Could not parse CAN Frame on: %s", err)
				continue
			}
			d.dispatchMessage(&StampedMessage{
				M:  m,
				ID: ID,
				T:  t,
			})
		}
	}
}

func (d *arkeDispatcher) Register(devicesID arke.NodeID) <-chan *StampedMessage {
	if d.channels == nil {
		panic("register on closed dispatcher")
	}

	d.mx.Lock()
	defer d.mx.Unlock()

	newChannel := make(chan *StampedMessage, 10)

	d.channels[int(devicesID)] = append(d.channels[int(devicesID)], newChannel)

	return newChannel
}

func (d *arkeDispatcher) Send(id arke.NodeID, m arke.SendableMessage) error {
	return arke.SendMessage(d.intf, m, false, id)
}

func (d *arkeDispatcher) Close() error {
	defer d.closeChannels()
	err := d.intf.Close()
	if d.done == nil {
		return err
	}
	<-d.done
	d.done = nil
	return err
}

func DispatchInterface(ifname string) (ArkeDispatcher, error) {
	intf, err := socketcan.NewRawInterface(ifname)
	if err != nil {
		return nil, err
	}
	return NewArkeDispatcher(ifname, intf), nil
}

func NewArkeDispatcher(ifname string, intf socketcan.RawInterface) ArkeDispatcher {
	return &arkeDispatcher{
		channels: make(map[int][]chan *StampedMessage),
		intf:     intf,
		logger:   log.New(os.Stderr, "[dispatch/"+ifname+"] ", 0),
	}
}
