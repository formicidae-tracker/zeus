package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"os"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/zeus"
	"github.com/grandcat/zeroconf"
)

type Zeus struct {
	logger         *log.Logger
	slcandManagers map[string]*SlcandManager
	quit, idle     chan struct{}

	zones map[string]ZoneDefinition

	busManagers map[string]BusManager
	interpolers map[string]*InterpolationManager
	reporters   map[string]*RPCReporter

	stop chan struct{}
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
		slcandManagers: make(map[string]*SlcandManager),
		zones:          c.Zones,
		busManagers:    make(map[string]BusManager),
		interpolers:    map[string]*InterpolationManager{},
		reporters:      map[string]*RPCReporter{},
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

	m, ok := z.busManagers[def.CANInterface]
	if ok == true {
		return m, nil
	}
	z.logger.Printf("Opening interface '%s'", def.CANInterface)
	intf, err := socketcan.NewRawInterface(def.CANInterface)
	if err != nil {
		return nil, err
	}
	b := NewBusManager(def.CANInterface, intf, zeus.HeartBeatPeriod)
	z.busManagers[def.CANInterface] = b
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

func (z *Zeus) setupZoneClimate(name string, zone zeus.Zone) error {
	_, err := z.managerForZone(name)
	if err != nil {
		return err
	}

	return fmt.Errorf("Not yet implemented")
}

func (z *Zeus) startClimate(season zeus.SeasonFile) error {
	if err := z.checkSeason(season); err != nil {
		return fmt.Errorf("invalid season file: %s", err)
	}

	for name, zone := range season.Zones {
		if err := z.setupZoneClimate(name, zone); err != nil {
			return fmt.Errorf("Could not setup zone '%s': %s", name, err)
		}
	}

	return fmt.Errorf("Not yet implemented")
}

func (z *Zeus) stopClimate() error {
	if z.stop == nil {
		return fmt.Errorf("Not running")
	}
	close(z.stop)
	z.waitClimate()
	z.stop = nil
	return nil
}

func (z *Zeus) waitClimate() {
	panic("NOT IMPLEMENTED")
}

func (z *Zeus) StartClimate(season zeus.SeasonFile, reply *error) error {
	*reply = z.startClimate(season)
	return nil
}

func (z *Zeus) StopClimate(ignored int, reply *error) error {
	*reply = z.stopClimate()
	return nil
}
