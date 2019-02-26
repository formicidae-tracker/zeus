package main

import (
	"sync"

	. "gopkg.in/check.v1"
)

type AlarmNotifierSuite struct {
}

var _ = Suite(&AlarmNotifierSuite{})

func (s *AlarmNotifierSuite) TestAlarmNotifier(c *C) {
	n := NewAlarmNotifier()
	subscribers := &sync.WaitGroup{}
	subscribing := &sync.WaitGroup{}
	dispatcher := &sync.WaitGroup{}
	go func() {
		dispatcher.Add(1)
		n.Dispatch()
		dispatcher.Done()
	}()

	for i := 0; i < 3; i++ {
		subscribers.Add(1)
		subscribing.Add(1)
		go func() {
			ca := n.Subscribe()
			subscribing.Done()
			c.Check(<-ca, Equals, WaterLevelCritical)
			c.Check(<-ca, Equals, TemperatureUnreachable)
			c.Check(<-ca, Equals, HumidityOutOfBound)
			subscribers.Done()
		}()
	}
	subscribing.Wait()
	n.Notify() <- WaterLevelCritical
	n.Notify() <- TemperatureUnreachable
	n.Notify() <- HumidityOutOfBound
	subscribers.Wait()
	n.Close()
	dispatcher.Wait()
}

func (s *AlarmNotifierSuite) TestAlarmNotifierHanging(c *C) {
	n := NewAlarmNotifier()
	subscribers := &sync.WaitGroup{}
	subscribing := &sync.WaitGroup{}
	dispatcher := &sync.WaitGroup{}

	go func() {
		dispatcher.Add(1)
		n.Dispatch()
		dispatcher.Done()
	}()

	subscribers.Add(1)
	subscribing.Add(1)
	go func() {
		ca := n.Subscribe()
		subscribing.Done()
		for i := 0; i < 11; i++ {
			_, ok := <-ca
			c.Check(ok, Equals, true)
		}
		_, ok := <-ca
		c.Check(ok, Equals, false)
		subscribers.Done()
	}()

	subscribers.Add(1)
	subscribing.Add(1)
	hanger := make(chan struct{})
	go func() {
		ca := n.Subscribe()
		subscribing.Done()
		_, ok := <-hanger
		c.Check(ok, Equals, false)
		for i := 0; i < 10; i++ {
			_, ok := <-ca
			c.Check(ok, Equals, true)
		}
		_, ok = <-ca
		c.Check(ok, Equals, false)
		subscribers.Done()
	}()

	subscribing.Wait()
	for i := 0; i < 11; i++ {
		n.Notify() <- WaterLevelUnreadable
	}
	close(hanger)
	n.Close()
	subscribers.Wait()
	dispatcher.Wait()

}
