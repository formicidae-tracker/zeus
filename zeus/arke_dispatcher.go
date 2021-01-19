package main

import (
	"fmt"
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
	Register(devicesID arke.NodeID) <-chan StampedMessage
	Close() error
}

type arkeDispatcher struct {
	channels map[int][]chan StampedMessage
	intf     socketcan.RawInterface
}

func (d *arkeDispatcher) Dispatch() {
	return
}

func (d *arkeDispatcher) Register(devicesID arke.NodeID) <-chan StampedMessage {
	return nil
}

func (d *arkeDispatcher) Close() error {
	return nil
}

func DispatchInterface(ifname string) (ArkeDispatcher, error) {
	return nil, fmt.Errorf("Not yet implemented")
}

func NewArkeDispatcher(intf socketcan.RawInterface) ArkeDispatcher {
	return &arkeDispatcher{
		channels: make(map[int][]chan StampedMessage),
		intf:     intf,
	}
}
