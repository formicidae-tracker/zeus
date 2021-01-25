package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/adrg/xdg"
	"github.com/formicidae-tracker/zeus"
)

type TimedAlarm struct {
	Alarm zeus.Alarm
	Time  time.Time
}

type AlarmMonitor interface {
	Name() string
	Monitor()
	Inbound() chan<- TimedAlarm
	Outbound() <-chan zeus.AlarmEvent
}

type alarmMonitor struct {
	inbound  chan TimedAlarm
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

	logfilename, err := xdg.DataFile("fort-experiments/climate-debug/box.txt")
	if err != nil {
		panic(err.Error())
	}

	logfile, err := os.Create(logfilename)
	if err != nil {
		panic(err.Error())
	}
	defer logfile.Close()

	for {
		select {
		case a, ok := <-m.inbound:
			if ok == false {
				close(quit)
				return
			}
			fmt.Fprintf(logfile, "%s: %+v\n", a.Time, a.Alarm)
			if t, ok := trigged[a.Alarm.Reason()]; ok == false || t <= 0 {
				go func() {
					m.outbound <- zeus.AlarmEvent{
						Reason:   a.Alarm.Reason(),
						Priority: a.Alarm.Priority(),
						Status:   zeus.AlarmOn,
						Time:     a.Time,
						Zone:     m.name,
					}
				}()
				trigged[a.Alarm.Reason()] = 1
				alarms[a.Alarm.Reason()] = a.Alarm
				fmt.Fprintf(logfile, "%s: %s:%d\n", time.Now(), a.Alarm.Reason(), 1)
			} else {
				trigged[a.Alarm.Reason()] += 1
				fmt.Fprintf(logfile, "%s: up %s:%d\n", time.Now(), a.Alarm.Reason(), trigged[a.Alarm.Reason()])
			}
			go wakeupAfter(clearWakeup, quit, a.Alarm.Reason(), 3*a.Alarm.RepeatPeriod())
		case r := <-clearWakeup:
			t, ok := trigged[r]
			fmt.Fprintf(logfile, "%s: down %s:%d\n", time.Now(), r, t)
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

func (m *alarmMonitor) Inbound() chan<- TimedAlarm {
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
		inbound:  make(chan TimedAlarm, 30),
		outbound: make(chan zeus.AlarmEvent, 60),
		name:     path.Join(hostname, "zone", zoneName),
	}, nil
}
