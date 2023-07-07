package main

import (
	"os/exec"
	"path"
	"sync"
	"syscall"
	"time"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/formicidae-tracker/olympus/pkg/tm"
	"github.com/sirupsen/logrus"
)

type StampedMessage struct {
	M  arke.ReceivableMessage
	T  time.Time
	ID arke.NodeID
}

type ArkeDispatcher interface {
	Dispatch(chan<- struct{})
	Register(devicesID arke.NodeID) <-chan *StampedMessage
	Name() string
	Interface() socketcan.RawInterface
	Close() error
}

type arkeDispatcher struct {
	mx       sync.RWMutex
	channels map[int][]chan *StampedMessage

	name   string
	intf   socketcan.RawInterface
	logger *logrus.Entry
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
		d.logger.WithFields(logrus.Fields{
			"ID":      m.ID,
			"message": m.M.String(),
		}).Warn("receiver not ready, dropping message")
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

func (d *arkeDispatcher) Dispatch(ready chan<- struct{}) {
	d.done = make(chan struct{})

	defer close(d.done)
	d.logger.Info("started")
	close(ready)
	for {
		f, err := d.intf.Receive()
		if err != nil {
			if errno, ok := err.(syscall.Errno); ok == true {
				if errno == syscall.EBADF || errno == syscall.ENETDOWN || errno == syscall.ENODEV {
					return
				}
			}
			d.logger.WithError(err).Error("could not receive CAN frame")
		} else {
			t := time.Now()
			m, ID, err := arke.ParseMessage(&f)
			if err != nil {
				d.logger.WithError(err).Error("could not parse CAN frame")
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
	defer func() {
		d.closeChannels()
		d.logger.Info("closed")
	}()

	d.logger.Debug("closing")
	err := d.intf.Close()
	if d.done == nil {
		return err
	}
	select {
	case <-d.done:
		d.done = nil
		return err
	case <-time.After(3 * time.Second):
		d.logger.Warn("dispatcher apparently hang up, sending request over bus")
		cmd := exec.Command("cansend", d.name, "007#")
		cmd.Run()
	}
	select {
	case <-d.done:
		d.done = nil
		return err
	case <-time.After(3 * time.Second):
		d.logger.Panic("closing hangup") // will panic
		return nil
	}
}

func DispatchInterface(ifname string) (ArkeDispatcher, error) {
	intf, err := socketcan.NewRawInterface(ifname)
	if err != nil {
		return nil, err
	}
	return NewArkeDispatcher(ifname, intf), nil
}

func (d *arkeDispatcher) Name() string {
	return d.name
}

func (d *arkeDispatcher) Interface() socketcan.RawInterface {
	return d.intf
}

func NewArkeDispatcher(ifname string, intf socketcan.RawInterface) ArkeDispatcher {
	return &arkeDispatcher{
		channels: make(map[int][]chan *StampedMessage),
		name:     ifname,
		intf:     intf,
		logger:   tm.NewLogger(path.Join("dispatch", ifname)),
	}
}
