package main

import (
	"os"
	"path"
	"time"
)

type AlarmStatus int

const (
	AlarmOn AlarmStatus = iota
	AlarmOff
)

type AlarmEvent struct {
	Zone   string
	Alarm  Alarm
	Status AlarmStatus
	Time   time.Time
}

type AlarmMonitor interface {
	Name() string
	Monitor()
	Inbound() chan<- Alarm
	Outbound() <-chan AlarmEvent
}

type alarmMonitor struct {
	inbound  chan Alarm
	outbound chan AlarmEvent
	name     string
}

func (m *alarmMonitor) Name() string {
	return m.name
}

func wakeupAfter(wakeup chan<- string, reason string, after time.Duration) {
	time.Sleep(after)
	wakeup <- reason
	defer func() {
		//silently recovering writing to a closed channel
		recover()
	}()
}

func (m *alarmMonitor) Monitor() {
	trigged := make(map[string]int)
	alarms := make(map[string]Alarm)
	clearWakeup := make(chan string)

	defer func() {
		close(m.outbound)
		close(clearWakeup)
	}()

	for {
		select {
		case a, ok := <-m.inbound:
			if ok == false {
				return
			}
			if t, ok := trigged[a.Reason()]; ok == false || t <= 0 {
				go func() {
					m.outbound <- AlarmEvent{
						Alarm:  a,
						Status: AlarmOn,
						Time:   time.Now(),
						Zone:   m.name,
					}
				}()
				trigged[a.Reason()] = 1
				alarms[a.Reason()] = a
			} else {
				trigged[a.Reason()] += 1
			}
			wakeupAfter(clearWakeup, a.Reason(), 3*a.RepeatInterval())
		case r := <-clearWakeup:
			t, ok := trigged[r]
			if ok == false {
				// should not happen but lets says it does
				continue
			}

			if t == 1 {
				go func() {
					m.outbound <- AlarmEvent{
						Alarm:  alarms[r],
						Status: AlarmOff,
						Time:   time.Now(),
						Zone:   m.name,
					}
				}()
			}
			if t != 0 {
				trigged[r] = t - 1
			}
		}
	}
}

func (m *alarmMonitor) Inbound() chan<- Alarm {
	return m.inbound
}

func (m *alarmMonitor) Outbound() <-chan AlarmEvent {
	return m.outbound
}

func NewAlarmMonitor(zoneName string) (AlarmMonitor, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &alarmMonitor{
		inbound:  make(chan Alarm, 30),
		outbound: make(chan AlarmEvent, 60),
		name:     path.Join(hostname, "zones", zoneName),
	}, nil
}
