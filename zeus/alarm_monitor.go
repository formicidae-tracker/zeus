package main

import (
	"fmt"
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
	inbound    chan zeus.Alarm
	outbound   chan zeus.AlarmEvent
	logger     *log.Logger
	concatened chan string
	name       string
}

func (m *alarmMonitor) Name() string {
	return m.name
}

type deadlineMeeter struct {
	deadlines map[string]time.Time
	timer     *time.Timer
}

func newDeadLineMeeter() *deadlineMeeter {
	return &deadlineMeeter{
		deadlines: make(map[string]time.Time),
		timer:     time.NewTimer(0),
	}
}

func (d *deadlineMeeter) next(now time.Time) <-chan time.Time {
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
	wait := minTime.Sub(now)
	if !d.timer.Stop() {
		//since we are waiting concurrently for popping, it may block
		//forever. So we must poll to drain the channel without
		//blocking, which may make a spurious fire. However pop will
		//just return an empty list.
		select {
		case <-d.timer.C:
		default:
		}
	}
	d.timer.Reset(wait)

	return d.timer.C
}

func (d *deadlineMeeter) pushDeadline(reason string, after time.Duration) <-chan time.Time {
	now := time.Now()
	d.deadlines[reason] = now.Add(after)
	return d.next(now)
}

func (d *deadlineMeeter) pop(now time.Time) ([]string, <-chan time.Time) {
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

func (m *alarmMonitor) concatenedPrintf(format string, args ...interface{}) {
	m.concatened <- fmt.Sprintf(format, args...)
}

func (m *alarmMonitor) logConcatened() {
	period := 10 * time.Minute
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	logs := make(map[string]int)
	for {
		select {
		case <-ticker.C:
			for l, count := range logs {
				m.logger.Printf("(%d times in %s) %s", count, period, l)
			}
			logs = make(map[string]int)
		case l, ok := <-m.concatened:
			if ok == false {
				return
			}
			logs[l] = logs[l] + 1
		}
	}

}

func (m *alarmMonitor) Monitor() {
	alarms := make(map[string]zeus.Alarm)

	defer func() {
		close(m.concatened)
		close(m.outbound)
	}()
	go m.logConcatened()
	quit := make(chan struct{})

	meeter := newDeadLineMeeter()

	var wakeUpChan <-chan time.Time = nil

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
						Reason:         a.Reason(),
						Flags:          a.Flags(),
						Status:         zeus.AlarmOn,
						Time:           time.Now(),
						ZoneIdentifier: m.name,
					}
				}()

			}
			alarms[a.Reason()] = a
			wakeUpChan = meeter.pushDeadline(a.Reason(), a.DeadLine())
		case now := <-wakeUpChan:
			var expired []string = nil
			expired, wakeUpChan = meeter.pop(now)

			if len(expired) == 0 {
				m.concatenedPrintf("spurious pop")
			}

			for _, r := range expired {
				a, ok := alarms[r]
				if ok == false {
					// should not happen but lets says it does
					continue
				}
				go func() {
					m.outbound <- zeus.AlarmEvent{
						Reason:         a.Reason(),
						Flags:          a.Flags(),
						Status:         zeus.AlarmOff,
						Time:           now,
						ZoneIdentifier: m.name,
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
		inbound:    make(chan zeus.Alarm, 30),
		outbound:   make(chan zeus.AlarmEvent, 60),
		name:       path.Join(hostname, "zone", zoneName),
		logger:     log.New(os.Stderr, "[zone/"+zoneName+"/alarm] ", 0),
		concatened: make(chan string),
	}, nil
}
