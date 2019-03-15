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
	socketcan "github.com/atuleu/golang-socketcan"
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
		if len(olympusHost) == 0 {
			return fmt.Errorf("Could not find an olympus host")
		}
		log.Printf("Will send all data to olympus at %s", olympusHost)

	}
	managers := map[string]BusManager{}
	rpc := map[string]*RPCReporter{}
	alarmMonitors := map[string]AlarmMonitor{}

	allCapabilities := []capability{}
	interpolers := map[string]*InterpolationManager{}
	init := make(chan struct{})
	quit := make(chan struct{})
	wgInterpolation := &sync.WaitGroup{}
	wgRpc := &sync.WaitGroup{}
	for zname, z := range c.Zones {
		logger := log.New(os.Stderr, "[zone/"+zname+"]: ", log.LstdFlags)

		logger.Printf("Loading zone")

		m, ok := managers[z.CANInterface]
		if ok == false {
			var err error
			logger.Printf("Opening interface '%s'", z.CANInterface)
			intf, err := socketcan.NewRawInterface(z.CANInterface)
			if err != nil {
				return err
			}

			managers[z.CANInterface] = NewBusManager(z.CANInterface, intf, dieu.HeartBeatPeriod)
		}

		var stateReports chan<- dieu.StateReport = nil

		reporters := []ClimateReporter{}
		if cmd.NoAvahi == false {
			rpc[zname], err = NewRPCReporter(zname, olympusHost, z)
			if err != nil {
				return err
			}
			reporters = append(reporters, rpc[zname])
			stateReports = rpc[zname].StateChannel()
			wgRpc.Add(1)
			go rpc[zname].Report(wgRpc)
		}

		capabilities, err := ComputeZoneRequirements(&z, reporters)
		if err != nil {
			return err
		}
		allCapabilities = append(allCapabilities, capabilities...)

		interpolers[zname], err = NewInterpolationManager(zname, z.States, z.Transitions, capabilities, stateReports)
		if err != nil {
			return err
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
				close(rpc[zname].AlarmChannel())
			}()
		} else {
			go func() {
				logger := log.New(os.Stderr, "[zone/"+zname+"/alarm]: ", log.LstdFlags)
				for ae := range alarmMonitors[zname].Outbound() {
					logger.Printf("%+v", ae)
				}
			}()
		}

		m.AssignCapabilitiesForID(arke.NodeID(z.DevicesID), capabilities, alarmMonitors[zname].Inbound())
		wgInterpolation.Add(1)
		go interpolers[zname].Interpolate(wgInterpolation, init, quit)
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
	close(quit) //will close interpolers
	wgInterpolation.Wait()
	for intfName, m := range managers {
		log.Printf("Closing interface '%s'", intfName)
		m.Close()
	}
	wgManager.Wait()

	for _, c := range allCapabilities {
		c.Close()
	}

	log.Printf("Waiting graceful exit")
	wgRpc.Wait()
	return nil
}

var runCommand = RunCommand{}

func init() {
	parser.AddCommand("run",
		"run the climate control",
		"run the climate control on the real hardware",
		&runCommand)
}
