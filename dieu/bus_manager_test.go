package main

import (
	"errors"
	"sync"
	"syscall"
	"testing"

	"git.tuleu.science/fort/dieu"
	"git.tuleu.science/fort/libarke/src-go/arke"
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

func (c *stubCapability) Action(s dieu.State) error {
	return c.zeus.SendMessage(&arke.ZeusSetPoint{
		Temperature: 24.0,
		Humidity:    50.0,
		Wind:        255,
	})
}

func (c *stubCapability) Callbacks() map[arke.MessageClass]callback {
	return map[arke.MessageClass]callback{
		arke.ZeusStatusMessage: func(alarms chan<- dieu.Alarm, m *StampedMessage) error {
			close(c.ready)
			<-c.release
			return nil
		},
	}
}

func (c *stubCapability) Close() error {
	return nil
}

func makeIDT(m arke.MessageClass, ID arke.NodeID) uint32 {
	return uint32(uint32(arke.StandardMessage)<<9 | uint32(m)<<3 | uint32(ID))
}

func TestBusManagerClose(t *testing.T) {
	s := BusManagerSuite{}
	s.SetUpMock(t)
	defer s.ctrl.Finish()

	manager := NewBusManager("mock", s.intf, dieu.HeartBeatPeriod)

	cap := &stubCapability{
		ready:   make(chan struct{}),
		release: make(chan struct{}),
	}

	alarms := make(chan dieu.Alarm)
	err := manager.AssignCapabilitiesForID(1, []capability{cap}, alarms)
	if err != nil {
		t.Error(err)
	}

	s.intf.EXPECT().Send(socketcan.CanFrame{455, 2, []byte{136, 19}, false, false})
	s.intf.EXPECT().Close().Return(nil)
	gomock.InOrder(
		s.intf.EXPECT().Receive().Return(
			socketcan.CanFrame{
				ID:       makeIDT(arke.ZeusStatusMessage, 1),
				Extended: false,
				RTR:      false,
				Dlc:      7,
				Data:     []byte{0, 0, 0, 0, 0, 0, 0, 0},
			}, nil),
		s.intf.EXPECT().Receive().Return(socketcan.CanFrame{}, nil),
		s.intf.EXPECT().Receive().Return(socketcan.CanFrame{}, errors.New("foo")),
		s.intf.EXPECT().Receive().Return(socketcan.CanFrame{}, syscall.EBADF),
	)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		manager.Listen()
		wg.Done()
	}()

	// //time.Sleep(20 * time.Millisecond)
	// select {
	// case <-alarms:
	// default:
	// 	t.Error("Should have received an alarm")
	// }

	closed := make(chan struct{})
	<-cap.ready
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

	_, ok := <-closed
	if ok != false {
		t.Error("Should be closed")
	}
}
