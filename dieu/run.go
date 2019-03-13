package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"git.tuleu.science/fort/libarke/src-go/arke"
	"github.com/grandcat/zeroconf"
)

type RunCommand struct {
}

func GetOlympusHost() (string, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return "", err
	}

	entries := make(chan *zeroconf.ServiceEntry)
	errors := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	go func() {
		err = resolver.Browse(ctx, "_olympus._tcp", "local.", entries)
		if err != nil {
			errors <- err
		}
		<-ctx.Done()
		close(errors)

	}()
	defer cancel()

	select {
	case e := <-entries:
		return fmt.Sprintf("%s:%d", strings.TrimSuffix(e.HostName, "."), e.Port), nil
	case err := <-errors:
		return "", err
	}
}

func (cmd *RunCommand) Execute(args []string) error {
	c, err := opts.LoadConfig()
	if err != nil {
		return err
	}

	olympusHost, err := GetOlympusHost()
	if err != nil {
		return err
	}
	log.Printf("Will send all data to olympus at %s", olympusHost)

	managers := map[string]BusManager{}
	rpc := map[string]*RPCReporter{}
	alarmMonitors := map[string]AlarmMonitor{}

	interpolers := map[string]ClimateInterpoler{}
	start := time.Now().UTC()
	init := make(chan struct{})
	for zname, z := range c.Zones {
		log.Printf("Loading zone '%s'", zname)
		log.Printf("[%s]: compute interpoler", zname)
		var err error
		interpolers[zname], err = NewClimateInterpoler(z.States, z.Transitions, start)
		if err != nil {
			return err
		}

		m, ok := managers[z.CANInterface]
		if ok == false {
			log.Printf("[%s]: opening %s", zname, z.CANInterface)
			var err error
			m, err = NewBusManager(z.CANInterface)
			if err != nil {
				return err
			}
			managers[z.CANInterface] = m
		}

		log.Printf("[%s]: opening RPC connection to %s", zname, olympusHost)
		rpc[zname], err = NewRPCReporter(zname, olympusHost)
		if err != nil {
			return err
		}
		go rpc[zname].Report()

		alarmMonitors[zname], err = NewAlarmMonitor(zname)
		if err != nil {
			return err
		}
		go alarmMonitors[zname].Monitor()
		go func() {
			for ae := range alarmMonitors[zname].Outbound() {
				rpc[zname].AlarmChannel() <- ae
			}
		}()

		capabilities, err := ComputeZoneRequirements(&z, []ClimateReporter{rpc[zname]})
		if err != nil {
			return err
		}

		m.AssignCapabilitiesForID(arke.NodeID(z.DevicesID), capabilities, alarmMonitors[zname].Inbound())

		go func(i ClimateInterpoler, caps []capability) {
			log.Printf("[%s]: Starting inetrpolation loop ", zname)
			quit := make(chan struct{})
			go func() {
				sigint := make(chan os.Signal, 1)
				signal.Notify(sigint, os.Interrupt)
				<-sigint
				close(quit)
			}()
			<-init
			now := time.Now()
			cur := i.CurrentInterpolation(now)
			for _, c := range caps {
				c.Action(cur.State(now))
			}

			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			for {
				select {
				case <-quit:
					return
				case <-timer.C:
					now := time.Now()
					cur := i.CurrentInterpolation(now)
					for _, c := range caps {
						c.Action(cur.State(now))
					}
				}
			}
		}(interpolers[zname], capabilities)
	}

	wg := sync.WaitGroup{}
	for _, m := range managers {
		wg.Add(1)
		go func() {
			log.Printf("Starting CAN loop")
			m.Listen()
			wg.Done()
		}()

	}

	close(init)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	<-sigint
	for _, m := range managers {
		log.Printf("closing interface")
		m.Close()
	}

	wg.Wait()

	return nil
}

var runCommand = RunCommand{}

func init() {
	parser.AddCommand("run",
		"run the climate control",
		"run the climate control on the real hardware",
		&runCommand)
}
