package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"sync"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/formicidae-tracker/zeus"
	"github.com/grandcat/zeroconf"
)

type ZoneRunner struct {
	interpoler   *InterpolationManager
	reporter     *RPCReporter
	capabilities []capability
	alarmMonitor AlarmMonitor
}

type Zeus struct {
	logger         *log.Logger
	slcandManagers map[string]*SlcandManager
	quit, idle     chan struct{}

	zones map[string]ZoneDefinition

	olympusHost string
	runners     map[string]*ZoneRunner
	managers    map[string]BusManager

	wgReporter, wgInterpolation, wgManager sync.WaitGroup

	stop, init chan struct{}
}

func (z *Zeus) openSlcands(interfaces map[string]string) error {
	for ifname, devname := range interfaces {
		z.logger.Printf("Starting slcand for '%s' on %s", ifname, devname)
		slcandManager, err := OpenSlcand(ifname, devname)
		if err != nil {
			return err
		}
		z.slcandManagers[ifname] = slcandManager
	}
	return nil
}

func (z *Zeus) closeSlcands() {
	for ifname, s := range z.slcandManagers {
		if err := s.Close(); err != nil {
			z.logger.Printf("could not close slcand '%s': %s", ifname, err)
		}
	}
}

func OpenZeus(c Config) (*Zeus, error) {
	if err := c.Check(); err != nil {
		return nil, fmt.Errorf("Invalid config: %s", err)
	}
	z := &Zeus{
		olympusHost:    c.Olympus,
		slcandManagers: make(map[string]*SlcandManager),
		zones:          c.Zones,
		runners:        make(map[string]*ZoneRunner),
		logger:         log.New(os.Stderr, "[zeus] ", 0),
	}
	if err := z.openSlcands(c.Interfaces); err != nil {
		return nil, err
	}

	return z, nil
}

func (z *Zeus) spawnZeroconf() {
	go func() {
		host, err := os.Hostname()
		if err != nil {
			z.logger.Printf("zeroconf error: could not get hostname: %s", err)
			return
		}
		server, err := zeroconf.Register("zeus."+host, "_leto._tcp", "local.", zeus.ZEUS_PORT, nil, nil)
		if err != nil {
			z.logger.Printf("zeroconf error: %s", err)
			return
		}
		<-z.idle
		server.Shutdown()
	}()
}

func (z *Zeus) runRPC() error {
	rpcRouter := rpc.NewServer()
	rpcRouter.Register(z)
	rpcRouter.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	rpcServer := http.Server{
		Addr:    fmt.Sprintf(":%d", zeus.ZEUS_PORT),
		Handler: rpcRouter,
	}

	go func() {
		<-z.quit
		if err := rpcServer.Shutdown(context.Background()); err != nil {
			z.logger.Printf("rpc shutdown error: %s", err)
		}
		close(z.idle)
	}()

	if err := rpcServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	<-z.idle
	z.quit = nil
	z.idle = nil
	return nil
}

func (z *Zeus) run() error {
	defer z.closeSlcands()
	z.quit = make(chan struct{})
	z.idle = make(chan struct{})
	z.spawnZeroconf()

	return z.runRPC()
}

func (z *Zeus) shutdown() error {
	if z.quit == nil {
		return fmt.Errorf("zeus: not started")
	}
	close(z.quit)
	return nil
}

func (z *Zeus) hasZone(name string) bool {
	_, ok := z.zones[name]
	return ok
}

func (z *Zeus) managerForZone(zoneName string) (BusManager, error) {
	def, ok := z.zones[zoneName]
	if ok == false {
		return nil, fmt.Errorf("Unknown zone '%s'", zoneName)
	}

	m, ok := z.managers[def.CANInterface]
	if ok == true {
		return m, nil
	}
	z.logger.Printf("Opening interface '%s'", def.CANInterface)
	intf, err := socketcan.NewRawInterface(def.CANInterface)
	if err != nil {
		return nil, err
	}
	b := NewBusManager(def.CANInterface, intf, zeus.HeartBeatPeriod)
	z.managers[def.CANInterface] = b
	return b, nil
}

func (z *Zeus) checkSeason(season zeus.SeasonFile) error {
	for zoneName, _ := range season.Zones {
		if z.hasZone(zoneName) {
			return fmt.Errorf("missing zone '%s'", zoneName)
		}
	}
	return nil
}

func (z *Zeus) setupZoneClimate(name string, zone zeus.Zone, devicesID arke.NodeID) (*ZoneRunner, error) {
	runner := &ZoneRunner{}
	manager, err := z.managerForZone(name)
	if err != nil {
		return nil, err
	}

	reporters := []ClimateReporter{}

	var stateReports chan<- zeus.StateReport = nil

	if len(z.olympusHost) != 0 {
		runner.reporter, err = NewRPCReporter(name, z.olympusHost, zone, os.Stderr)
		if err != nil {
			return nil, err
		}
		reporters = append(reporters, runner.reporter)
		stateReports = runner.reporter.StateChannel()
	}

	runner.capabilities, err = ComputeZoneRequirements(&zone, reporters)
	if err != nil {
		return nil, err
	}

	runner.interpoler, err = NewInterpolationManager(name, zone.States, zone.Transitions, runner.capabilities, stateReports, os.Stderr)
	if err != nil {
		return nil, err
	}

	runner.alarmMonitor, err = NewAlarmMonitor(name)
	if err != nil {
		return nil, err
	}

	manager.AssignCapabilitiesForID(devicesID, runner.capabilities, runner.alarmMonitor.Inbound())
	return runner, err
}

func (z *Zeus) spawnReporter(reporter *RPCReporter) {
	if reporter == nil {
		return
	}
	z.wgReporter.Add(1)
	go reporter.Report(&z.wgReporter)
}

func spawnAlarmMonitor(name string, alarmMonitor AlarmMonitor, reporter *RPCReporter) {
	go alarmMonitor.Monitor()
	if reporter != nil {
		go func() {
			for event := range alarmMonitor.Outbound() {
				reporter.AlarmChannel() <- event
			}
			close(reporter.AlarmChannel())
		}()
	} else {
		go func() {
			logger := log.New(os.Stderr, "[zone/"+name+"/alarm] ", 0)
			for event := range alarmMonitor.Outbound() {
				logger.Printf("%+v", event)
			}
		}()
	}
}

func (z *Zeus) spawnInterpoler(interpoler *InterpolationManager) {
	z.wgInterpolation.Add(1)
	go interpoler.Interpolate(&z.wgInterpolation, z.init, z.quit)
}

func (z *Zeus) spawnManager(manager BusManager) {
	z.wgManager.Add(1)
	go func() {
		manager.Listen()
		z.wgManager.Done()
	}()
}

func (z *Zeus) spawn(name string, runner *ZoneRunner) {
	z.spawnReporter(runner.reporter)
	spawnAlarmMonitor(name, runner.alarmMonitor, runner.reporter)

	z.spawnInterpoler(runner.interpoler)
}

func (z *Zeus) startClimate(season zeus.SeasonFile) error {
	if z.stop != nil {
		return fmt.Errorf("Already started")
	}

	if err := z.checkSeason(season); err != nil {
		return fmt.Errorf("invalid season file: %s", err)
	}

	for name, zone := range season.Zones {
		runner, err := z.setupZoneClimate(name, zone, arke.NodeID(z.zones[name].DevicesID))
		if err != nil {

			return fmt.Errorf("Could not setup zone '%s': %s", name, err)
		}

		z.runners[name] = runner
	}

	z.stop = make(chan struct{})
	z.init = make(chan struct{})

	for name, runner := range z.runners {
		z.logger.Printf("starting zone %s", name)
		z.spawn(name, runner)
	}

	for _, manager := range z.managers {
		z.spawnManager(manager)
	}

	close(z.init)
	z.init = nil
	return nil
}

func (z *Zeus) waitClimate() {
	z.wgInterpolation.Wait()
	for ifname, manager := range z.managers {
		z.logger.Printf("closing interface %s", ifname)
		manager.Close()
	}
	z.wgManager.Wait()
	for _, runner := range z.runners {
		for _, c := range runner.capabilities {
			c.Close()
		}
	}
	z.wgReporter.Wait()
}

func (z *Zeus) resetClimate() {
	z.quit = nil
	z.init = nil
	z.runners = make(map[string]*ZoneRunner)
	z.managers = make(map[string]BusManager)

}

func (z *Zeus) stopClimate() error {
	if z.stop == nil {
		return fmt.Errorf("Not running")
	}
	close(z.stop)
	z.waitClimate()
	z.resetClimate()
	return nil
}

func (z *Zeus) StartClimate(season zeus.SeasonFile, reply *error) error {
	*reply = z.startClimate(season)
	return nil
}

func (z *Zeus) StopClimate(ignored int, reply *error) error {
	*reply = z.stopClimate()
	return nil
}
