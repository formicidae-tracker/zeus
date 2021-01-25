package main

import (
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

type deadlineMeeter struct {
	deadlines map[string]time.Time
}

func (d *deadlineMeeter) next(now time.Time) *time.Timer {
	if len(d.deadlines) == 0 {
		return nil
	}
	set := false
	var minTime time.Time
	for _, when := range d.deadlines {
		if set == false || when.Before(minTime) == true {
			minTime = when
			set = true
		}
	}

	return time.NewTimer(minTime.Sub(now))
}

func (d *deadlineMeeter) pushDeadline(reason string, after time.Duration) *time.Timer {
	now := time.Now()
	d.deadlines[reason] = now.Add(after)
	return d.next(now)
}

func (d *deadlineMeeter) pop(now time.Time) ([]string, *time.Timer) {
	res := make([]string, 0, len(d.deadlines))
	for reason, when := range d.deadlines {
		if when.After(now) == true {
			continue
		}
		res = append(res, reason)
	}
	for _, reason := range res {
		delete(d.deadlines, reason)
	}
	return res, d.next(now)
}

func (m *alarmMonitor) Monitor() {
	alarms := make(map[string]zeus.Alarm)

	defer func() {
		close(m.outbound)
	}()

	quit := make(chan struct{})

	meeter := &deadlineMeeter{make(map[string]time.Time)}

	var wakeUpTimer *time.Timer = nil
	var wakeUpChan <-chan time.Time = nil

	pushTimer := func(newTimer *time.Timer) {
		if wakeUpTimer != nil {
			wakeUpTimer.Stop()
		}
		wakeUpTimer = newTimer
		if wakeUpTimer == nil {
			wakeUpChan = nil
		} else {
			wakeUpChan = wakeUpTimer.C
		}
	}

	for {
		select {
		case a, ok := <-m.inbound:
			if ok == false {
				close(quit)
				return
			}
			if _, ok := alarms[a.Reason()]; ok == false {
				go func() {
					m.outbound <- zeus.AlarmEvent{
						Reason: a.Reason(),
						Flags:  a.Flags(),
						Status: zeus.AlarmOn,
						Time:   time.Now(),
						Zone:   m.name,
					}
				}()

			}
			alarms[a.Reason()] = a
			pushTimer(meeter.pushDeadline(a.Reason(), a.DeadLine()))
		case now := <-wakeUpChan:
			expired, newTimer := meeter.pop(now)
			pushTimer(newTimer)

			for _, r := range expired {
				a, ok := alarms[r]
				if ok == false {
					// should not happen but lets says it does
					continue
				}
				go func() {
					m.outbound <- zeus.AlarmEvent{
						Reason: a.Reason(),
						Flags:  a.Flags(),
						Status: zeus.AlarmOff,
						Time:   now,
						Zone:   m.name,
					}
				}()
				delete(alarms, r)
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
