package main

import (
	"container/heap"
	"os"
	"path"
	"time"

	"github.com/formicidae-tracker/olympus/pkg/tm"
	"github.com/formicidae-tracker/zeus/internal/zeus"
	"github.com/sirupsen/logrus"
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
	logger     *logrus.Entry
	concatened chan string
	name       string

	stagged, fired            map[string]zeus.Alarm
	toDismiss, toFire, toKill alarmQueue
}

type alarmItem struct {
	name     string
	deadline time.Time
}

type alarmQueue []*alarmItem

func (q alarmQueue) Len() int {
	return len(q)
}

func (q alarmQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q alarmQueue) Less(i, j int) bool {
	return q[i].deadline.Before(q[j].deadline)
}

func (q *alarmQueue) Push(x any) {
	item := x.(*alarmItem)
	*q = append(*q, item)
}

func (q *alarmQueue) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*q = old[:n-1]
	return item
}

var infinity = time.Unix(1<<63-62135596802, 0)

func (q alarmQueue) Next() time.Time {
	if len(q) == 0 {
		return infinity
	}
	return q[0].deadline
}

func (q *alarmQueue) Update(a string, deadline time.Time) {
	i := 0
	var item *alarmItem
	for i, item = range *q {
		if item.name == a {
			break
		}
	}
	if i >= len(*q) {
		return
	}
	item.deadline = deadline
	heap.Fix(q, i)
}

func (m *alarmMonitor) Name() string {
	return m.name
}

func (m *alarmMonitor) Monitor() {
	defer func() {
		close(m.concatened)
		close(m.outbound)
	}()

	var timer <-chan time.Time

	for {
		select {
		case a, ok := <-m.inbound:
			if ok == false {
				return
			}
			now := time.Now()
			if _, ok := m.stagged[a.Identifier()]; ok == true {
				m.updateStagged(a, now)
			} else if _, ok := m.fired[a.Identifier()]; ok == true {
				m.updateFired(a, now)
			} else {
				m.stage(a, now)
			}
			timer = m.getNextDeadline(now)
		case now := <-timer:
			m.dismissAny(now)
			m.fireAny(now)
			m.killAny(now)
			timer = m.getNextDeadline(now)
		}
	}
}

func (m *alarmMonitor) getNextDeadline(now time.Time) <-chan time.Time {
	deadline := m.toDismiss.Next()
	if m.toFire.Next().Before(deadline) {
		deadline = m.toFire.Next()
	}
	if m.toKill.Next().Before(deadline) {
		deadline = m.toKill.Next()
	}

	if deadline.Equal(infinity) {
		return nil
	}

	return time.After(deadline.Sub(now))
}

func (m *alarmMonitor) updateStagged(alarm zeus.Alarm, now time.Time) {
	m.toDismiss.Update(alarm.Identifier(), now.Add(alarm.MinDownTime()))
}

func (m *alarmMonitor) updateFired(alarm zeus.Alarm, now time.Time) {
	m.toKill.Update(alarm.Identifier(), now.Add(alarm.MinDownTime()))
}

func (m *alarmMonitor) stage(a zeus.Alarm, now time.Time) {
	m.stagged[a.Identifier()] = a
	heap.Push(&m.toDismiss, &alarmItem{
		name:     a.Identifier(),
		deadline: now.Add(a.MinDownTime()),
	})
	heap.Push(&m.toFire, &alarmItem{
		name:     a.Identifier(),
		deadline: now.Add(a.MinUpTime()),
	})

	if a.Flags()&zeus.AdminOnly != 0 {
		return
	}

	adminAlarm := zeus.NewAlarmString(
		a.Flags()|zeus.AdminOnly,
		path.Join("admin", a.Identifier()),
		a.Description(),
		2*time.Second,
		2*time.Second,
	)

	m.stage(adminAlarm, now)
}

func (m *alarmMonitor) dismissAny(now time.Time) {
	for m.toDismiss.Next().Before(now) {
		item := heap.Pop(&m.toDismiss).(*alarmItem)
		// will do nothing if already fired
		delete(m.stagged, item.name)
	}
}

func (m *alarmMonitor) fireAny(now time.Time) {
	for m.toFire.Next().Before(now) {
		item := heap.Pop(&m.toFire).(*alarmItem)
		alarm, ok := m.stagged[item.name]
		if ok == false {
			//likely dismissed before, simply continue
			continue
		}
		delete(m.stagged, alarm.Identifier())
		m.outbound <- zeus.AlarmEvent{
			ZoneIdentifier: m.name,
			Identifier:     alarm.Identifier(),
			Description:    alarm.Description(),
			Flags:          alarm.Flags(),
			Status:         zeus.AlarmOn,
			Time:           item.deadline.Add(-1 * alarm.MinUpTime()),
		}
		heap.Push(&m.toKill, &alarmItem{
			name:     alarm.Identifier(),
			deadline: now.Add(alarm.MinDownTime())})
		m.fired[alarm.Identifier()] = alarm
	}
}

func (m *alarmMonitor) killAny(now time.Time) {
	for m.toKill.Next().Before(now) {
		item := heap.Pop(&m.toKill).(*alarmItem)
		alarm, ok := m.fired[item.name]
		if ok == false {
			// should never be reached, bu who knows
			continue
		}
		delete(m.fired, item.name)
		m.outbound <- zeus.AlarmEvent{
			ZoneIdentifier: m.name,
			Identifier:     alarm.Identifier(),
			Description:    alarm.Description(),
			Flags:          alarm.Flags(),
			Status:         zeus.AlarmOff,
			Time:           item.deadline.Add(-1 * alarm.MinUpTime()),
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
		logger:     tm.NewLogger(path.Join("zone", zoneName, "alarm")),
		concatened: make(chan string),
		fired:      make(map[string]zeus.Alarm),
		stagged:    make(map[string]zeus.Alarm),
	}, nil
}
