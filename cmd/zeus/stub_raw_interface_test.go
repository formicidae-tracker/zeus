package main

import (
	"syscall"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/libarke/src-go/arke"
)

type StubRawInterface struct {
	queue chan socketcan.CanFrame
}

func (i *StubRawInterface) Send(f socketcan.CanFrame) error {
	if i.isClosed() == true {
		return i.closedError()
	}
	return nil
}

func (i *StubRawInterface) closedError() error {
	return syscall.EBADF
}

func (i *StubRawInterface) isClosed() bool {
	return i.queue == nil
}

func (i *StubRawInterface) Receive() (socketcan.CanFrame, error) {
	if i.isClosed() == true {
		return socketcan.CanFrame{}, i.closedError()
	}
	f, ok := <-i.queue
	if ok == false {
		return socketcan.CanFrame{}, i.closedError()
	}
	return f, nil
}

func makeCANIDT(t arke.MessageType, c arke.MessageClass, n arke.NodeID) uint32 {
	return uint32((uint32(t) << 9) | (uint32(c) << 3) | uint32(n))
}

func (i *StubRawInterface) enqueue(m arke.SendableMessage, id arke.NodeID) {

	f := socketcan.CanFrame{
		ID:       makeCANIDT(arke.StandardMessage, m.MessageClassID(), id),
		Extended: false,
		RTR:      false,
		Data:     make([]byte, 8),
	}
	dlc, err := m.Marshal(f.Data)
	if err != nil {
		panic(err.Error())
	}
	f.Dlc = uint8(dlc)

	i.queue <- f
}

func (i *StubRawInterface) Close() error {
	if i.isClosed() == true {
		return nil
	}
	close(i.queue)
	i.queue = nil
	return nil
}

func NewStubRawInterface() *StubRawInterface {
	return &StubRawInterface{
		queue: make(chan socketcan.CanFrame),
	}
}
