package main

import (
	"bytes"
	"errors"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/formicidae-tracker/zeus"
	"github.com/formicidae-tracker/libarke/src-go/arke"
	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/golang/mock/gomock"
	. "gopkg.in/check.v1"
)

type BusManagerSuite struct {
	ctrl *gomock.Controller
	intf *MockRawInterface
}

var _ = Suite(&BusManagerSuite{})

func (s *BusManagerSuite) SetUpMock(t *testing.T) {
	s.ctrl = gomock.NewController(t)
	s.intf = NewMockRawInterface(s.ctrl)
}

type stubCapability struct {
	zeus    *Device
	ready   chan struct{}
	release chan struct{}
}

func (c *stubCapability) Requirements() []arke.NodeClass {
	return []arke.NodeClass{arke.ZeusClass}
}

func (c *stubCapability) SetDevices(devices map[arke.NodeClass]*Device) {
	c.zeus = devices[arke.ZeusClass]
}

func (c *stubCapability) Action(s zeus.State) error {
	return c.zeus.SendMessage(&arke.ZeusSetPoint{
		Temperature: 24.0,
		Humidity:    50.0,
		Wind:        255,
	})
}

func (c *stubCapability) Callbacks() map[arke.MessageClass]callback {
	return map[arke.MessageClass]callback{
		arke.ZeusStatusMessage: func(alarms chan<- zeus.Alarm, m *StampedMessage) error {
			close(c.ready)
			<-c.release
			return nil
		},
	}
}

func (c *stubCapability) Close() error {
	return nil
}

func makeIDT(t arke.MessageType, m arke.MessageClass, ID arke.NodeID) uint32 {
	return uint32(uint32(t)<<9 | uint32(m)<<3 | uint32(ID))
}

func TestBusManagerClose(t *testing.T) {
	s := BusManagerSuite{}
	s.SetUpMock(t)
	defer s.ctrl.Finish()

	manager := NewBusManager("mock", s.intf, 5*time.Millisecond)
	//removes all the nasty logs
	manager.(*busManager).log.SetOutput(bytes.NewBuffer([]byte{}))

	cap := &stubCapability{
		ready:   make(chan struct{}),
		release: make(chan struct{}),
	}

	alarms := make(chan zeus.Alarm, 10)
	err := manager.AssignCapabilitiesForID(1, []capability{cap}, alarms)
	if err != nil {
		t.Error(err)
	}

	err = manager.AssignCapabilitiesForID(1, []capability{}, alarms)
	if err == nil {
		t.Error("Should be able to assign ID only once")
	}

	closedInterface := make(chan struct{})

	s.intf.EXPECT().Send(socketcan.CanFrame{455, 2, []byte{5, 0}, false, false}).AnyTimes()
	s.intf.EXPECT().Close().AnyTimes().DoAndReturn(func() error {
		close(closedInterface)
		return nil
	})
	gomock.InOrder(
		s.intf.EXPECT().Receive().Return(
			socketcan.CanFrame{
				ID:       makeIDT(arke.HeartBeat, arke.ZeusStatusMessage, 1),
				Extended: false,
				RTR:      false,
				Dlc:      7,
				Data:     []byte{0, 0, 0, 0, 0, 0, 0, 0},
			}, nil),
		s.intf.EXPECT().Receive().Return(
			socketcan.CanFrame{
				ID:       makeIDT(arke.StandardMessage, arke.ZeusStatusMessage, 1),
				Extended: false,
				RTR:      false,
				Dlc:      7,
				Data:     []byte{0, 0, 0, 0, 0, 0, 0, 0},
			}, nil),
		s.intf.EXPECT().Receive().Return(socketcan.CanFrame{}, nil),
		s.intf.EXPECT().Receive().Return(socketcan.CanFrame{}, errors.New("foo")),
		s.intf.EXPECT().Receive().AnyTimes().DoAndReturn(func() (socketcan.CanFrame, error) {
			<-closedInterface
			return socketcan.CanFrame{}, syscall.EBADF
		}),
	)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		manager.Listen()
		wg.Done()
	}()

	closed := make(chan struct{})
	<-cap.ready

	a, ok := <-alarms
	if ok == true {
		if _, ok := a.(zeus.MissingDeviceAlarm); ok == false {
			t.Error("Should have received a missing device alarm")
		}
	}
	go func() {
		manager.Close()
		close(closed)
	}()

	select {
	case <-closed:
		t.Error("Should not be closed, callback is running")
	default:
	}
	close(cap.release)

	_, ok = <-closed
	if ok != false {
		t.Error("Should be closed")
	}

}
