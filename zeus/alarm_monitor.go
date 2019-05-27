package main

import (
	"log"
	"os"
	"path"
	"time"

	"github.com/formicidae-tracker/zeus"
)

type AlarmMonitor interface {
	Name() string
	Monitor()
	Inbound() chan<- zeus.Alarm
	Outbound() <-chan zeus.AlarmEvent
}

type alarmMonitor struct {
	inbound  chan zeus.Alarm
	outbound chan zeus.AlarmEvent
	name     string
}

func (m *alarmMonitor) Name() string {
	return m.name
}

func wakeupAfter(wakeup chan<- string, quit <-chan struct{}, reason string, after time.Duration) {
	time.Sleep(after)
	//we allow sending on closed wakeup channel even if we really try not to.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered: %s", r)
		}
	}()
	select {
	case <-quit:
		return
	default:
		wakeup <- reason
	}

}

func (m *alarmMonitor) Monitor() {
	trigged := make(map[string]int)
	alarms := make(map[string]zeus.Alarm)
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
					m.outbound <- zeus.AlarmEvent{
						Reason:   a.Reason(),
						Priority: a.Priority(),
						Status:   zeus.AlarmOn,
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
					m.outbound <- zeus.AlarmEvent{
						Reason:   alarms[r].Reason(),
						Priority: alarms[r].Priority(),
						Status:   zeus.AlarmOff,
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

func (m *alarmMonitor) Inbound() chan<- zeus.Alarm {
	return m.inbound
}

func (m *alarmMonitor) Outbound() <-chan zeus.AlarmEvent {
	return m.outbound
}

func NewAlarmMonitor(zoneName string) (AlarmMonitor, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &alarmMonitor{
		inbound:  make(chan zeus.Alarm, 30),
		outbound: make(chan zeus.AlarmEvent, 60),
		name:     path.Join(hostname, "zone", zoneName),
	}, nil
}
