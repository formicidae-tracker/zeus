package main

import (
	"os"
	"path"
	"sort"
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

type deadline struct {
	reason string
	when   time.Time
}

type deadlineList []deadline

func (l deadlineList) Len() int {
	return len(l)
}
func (l deadlineList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
func (l deadlineList) Less(i, j int) bool {
	return l[i].when.Before(l[j].when)
}

type deadlineMeeter struct {
	deadlines []deadline
}

func (d *deadlineMeeter) pushDeadline(reason string, after time.Duration) <-chan time.Time {
	now := time.Now()
	d.deadlines = append(d.deadlines, deadline{reason: reason, when: now.Add(after)})
	sort.Sort(deadlineList(d.deadlines))
	return time.After(d.deadlines[0].when.Sub(now) + 10*time.Millisecond)
}

func (d *deadlineMeeter) pop(now time.Time) ([]string, <-chan time.Time) {
	res := make([]string, 0, len(d.deadlines))
	i := 0
	deadline := deadline{}
	for i, deadline = range d.deadlines {
		if deadline.when.After(now) == true {
			break
		}
		res = append(res, deadline.reason)
	}
	d.deadlines = d.deadlines[i:]
	if len(d.deadlines) == 0 {
		return res, nil
	}
	return res, time.After(d.deadlines[0].when.Sub(now) + 10*time.Millisecond)
}

func (m *alarmMonitor) Monitor() {
	type alarmData struct {
		alarm zeus.Alarm
		count int
	}
	alarms := make(map[string]*alarmData)

	defer func() {
		close(m.outbound)
	}()

	quit := make(chan struct{})

	meeter := &deadlineMeeter{}

	var wakeUp <-chan time.Time = nil

	for {
		select {
		case a, ok := <-m.inbound:
			if ok == false {
				close(quit)
				return
			}
			if d, ok := alarms[a.Reason()]; ok == false || d.count <= 0 {
				go func() {
					m.outbound <- zeus.AlarmEvent{
						Reason: a.Reason(),
						Flags:  a.Flags(),
						Status: zeus.AlarmOn,
						Time:   time.Now(),
						Zone:   m.name,
					}
				}()
				alarms[a.Reason()] = &alarmData{alarm: a, count: 1}
			} else {
				alarms[a.Reason()].count += 1
			}
			wakeUp = meeter.pushDeadline(a.Reason(), a.DeadLine())
		case now := <-wakeUp:
			var expired []string = nil
			expired, wakeUp = meeter.pop(now)
			for _, r := range expired {
				a, ok := alarms[r]
				if ok == false {
					// should not happen but lets says it does
					continue
				}

				if a.count == 1 {
					go func() {
						m.outbound <- zeus.AlarmEvent{
							Reason: a.alarm.Reason(),
							Flags:  a.alarm.Flags(),
							Status: zeus.AlarmOff,
							Time:   now,
							Zone:   m.name,
						}
					}()
					delete(alarms, r)
				}
				if a.count > 1 {
					a.count -= 1
				}
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
