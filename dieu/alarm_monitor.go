package main

import (
	"os"
	"path"
	"time"

	"git.tuleu.science/fort/dieu"
)

type AlarmMonitor interface {
	Name() string
	Monitor()
	Inbound() chan<- dieu.Alarm
	Outbound() <-chan dieu.AlarmEvent
}

type alarmMonitor struct {
	inbound  chan dieu.Alarm
	outbound chan dieu.AlarmEvent
	name     string
}

func (m *alarmMonitor) Name() string {
	return m.name
}

func wakeupAfter(wakeup chan<- string, quit <-chan struct{}, reason string, after time.Duration) {
	time.Sleep(after)
	select {
	case <-quit:
		return
	default:
	}
	wakeup <- reason
}

func (m *alarmMonitor) Monitor() {
	trigged := make(map[string]int)
	alarms := make(map[string]dieu.Alarm)
	clearWakeup := make(chan string)

	defer func() {
		close(m.outbound)
		close(clearWakeup)
	}()

	quit := make(chan struct{})

	for {
		select {
		case a, ok := <-m.inbound:
			if ok == false {
				close(quit)
				return
			}
			if t, ok := trigged[a.Reason()]; ok == false || t <= 0 {
				go func() {
					m.outbound <- dieu.AlarmEvent{
						Reason:   a.Reason(),
						Priority: a.Priority(),
						Status:   dieu.AlarmOn,
						Time:     time.Now(),
						Zone:     m.name,
					}
				}()
				trigged[a.Reason()] = 1
				alarms[a.Reason()] = a
			} else {
				trigged[a.Reason()] += 1
			}
			go wakeupAfter(clearWakeup, quit, a.Reason(), 3*a.RepeatPeriod())
		case r := <-clearWakeup:
			t, ok := trigged[r]
			if ok == false {
				// should not happen but lets says it does
				continue
			}

			if t == 1 {
				go func() {
					m.outbound <- dieu.AlarmEvent{
						Reason:   alarms[r].Reason(),
						Priority: alarms[r].Priority(),
						Status:   dieu.AlarmOff,
						Time:     time.Now(),
						Zone:     m.name,
					}
				}()
			}
			if t != 0 {
				trigged[r] = t - 1
			}
		}
	}
}

func (m *alarmMonitor) Inbound() chan<- dieu.Alarm {
	return m.inbound
}

func (m *alarmMonitor) Outbound() <-chan dieu.AlarmEvent {
	return m.outbound
}

func NewAlarmMonitor(zoneName string) (AlarmMonitor, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &alarmMonitor{
		inbound:  make(chan dieu.Alarm, 30),
		outbound: make(chan dieu.AlarmEvent, 60),
		name:     path.Join(hostname, "zone", zoneName),
	}, nil
}
