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

	"git.tuleu.science/fort/dieu"
	"git.tuleu.science/fort/libarke/src-go/arke"
	"github.com/grandcat/zeroconf"
)

type RunCommand struct {
	NoAvahi bool `long:"no-olympus" short:"n" description:"Do not connect to olympus"`
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

	var olympusHost string
	if cmd.NoAvahi == false {
		olympusHost, err = GetOlympusHost()
		if err != nil {
			return err
		}
		log.Printf("Will send all data to olympus at %s", olympusHost)
	}
	managers := map[string]BusManager{}
	rpc := map[string]*RPCReporter{}
	alarmMonitors := map[string]AlarmMonitor{}

	interpolers := map[string]ClimateInterpoler{}
	start := time.Now().UTC()
	init := make(chan struct{})
	wgInterpolation := &sync.WaitGroup{}
	for zname, z := range c.Zones {
		log.Printf("Loading zone '%s'", zname)
		logger := log.New(os.Stderr, "[zone/"+zname+"/climate]: ", log.LstdFlags)
		logger.Printf("Compute Climate Interpoler")
		var err error
		interpolers[zname], err = NewClimateInterpoler(z.States, z.Transitions, start)
		if err != nil {
			return err
		}

		m, ok := managers[z.CANInterface]
		if ok == false {
			var err error
			m, err = NewBusManager(z.CANInterface)
			if err != nil {
				return err
			}
			managers[z.CANInterface] = m
		}

		reporters := []ClimateReporter{}
		if cmd.NoAvahi == false {
			rpc[zname], err = NewRPCReporter(zname, olympusHost, z)
			if err != nil {
				return err
			}
			reporters = append(reporters, rpc[zname])
			go rpc[zname].Report()
		}

		capabilities, err := ComputeZoneRequirements(&z, reporters)
		if err != nil {
			return err
		}

		alarms := []dieu.Alarm{}
		for _, c := range capabilities {
			alarms = append(alarms, c.Alarms()...)
		}

		alarmMonitors[zname], err = NewAlarmMonitor(zname)
		if err != nil {
			return err
		}
		go alarmMonitors[zname].Monitor()
		if cmd.NoAvahi == false {
			go func() {
				for ae := range alarmMonitors[zname].Outbound() {
					rpc[zname].AlarmChannel() <- ae
				}
			}()
		} else {
			go func() {
				for ae := range alarmMonitors[zname].Outbound() {
					log.Printf("%+v", ae)
				}
			}()
		}

		m.AssignCapabilitiesForID(arke.NodeID(z.DevicesID), capabilities, alarmMonitors[zname].Inbound())
		wgInterpolation.Add(1)
		go func(i ClimateInterpoler, caps []capability) {
			defer wgInterpolation.Done()
			log.Printf("[%s]: Starting interpolation loop ", zname)
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
			logger.Printf("Starting interpolation is %s", cur)

			timer := time.NewTicker(10 * time.Second)
			defer timer.Stop()
			for {
				select {
				case <-quit:
					logger.Printf("Closing climate interpolation")
					return
				case <-timer.C:
					now := time.Now()
					new := i.CurrentInterpolation(now)
					if cur != new {
						logger.Printf("New interpolation %s", new)
						cur = new
					}
					for _, c := range caps {
						c.Action(cur.State(now))
					}
				}
			}
		}(interpolers[zname], capabilities)
	}

	wgManager := &sync.WaitGroup{}
	for _, m := range managers {
		wgManager.Add(1)
		go func() {
			m.Listen()
			wgManager.Done()
		}()

	}

	close(init)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	<-sigint
	wgInterpolation.Wait()
	for intfName, m := range managers {
		log.Printf("Closing interface '%s'", intfName)
		m.Close()
	}

	for zname, r := range rpc {
		log.Printf("Closing rpc connection for '%s'", zname)
		close(r.ReportChannel())
	}

	log.Printf("Waiting graceful exit")
	wgManager.Wait()

	return nil
}

var runCommand = RunCommand{}

func init() {
	parser.AddCommand("run",
		"run the climate control",
		"run the climate control on the real hardware",
		&runCommand)
}
