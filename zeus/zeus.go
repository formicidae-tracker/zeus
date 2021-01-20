package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/zeus"
	"github.com/grandcat/zeroconf"
)

type Zeus struct {
	intfFactory func(ifname string) (socketcan.RawInterface, error)

	logger *log.Logger

	olympusHost string
	definitions map[string]ZoneDefinition

	dispatchers map[string]ArkeDispatcher
	runners     map[string]ZoneClimateRunner

	mx               sync.RWMutex
	quit, done, idle chan struct{}
}

func OpenZeus(c Config) (*Zeus, error) {
	if err := c.Check(); err != nil {
		return nil, fmt.Errorf("Invalid config: %s", err)
	}
	z := &Zeus{
		intfFactory: socketcan.NewRawInterface,
		logger:      log.New(os.Stderr, "[zeus] ", 0),
		olympusHost: c.Olympus,
		definitions: c.Zones,
		runners:     make(map[string]ZoneClimateRunner),
		dispatchers: make(map[string]ArkeDispatcher),
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
		server, err := zeroconf.Register("zeus."+host, "_zeus._tcp", "local.", zeus.ZEUS_PORT, nil, nil)
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
	z.quit = make(chan struct{})
	z.done = make(chan struct{})
	z.idle = make(chan struct{})
	defer close(z.done)
	for name, def := range z.definitions {
		z.logger.Printf("will manage zone '%s' on %s:%d", name, def.CANInterface, def.DevicesID)
	}

	z.spawnZeroconf()

	return z.runRPC()
}

func (z *Zeus) shutdown() error {
	if z.quit == nil {
		return fmt.Errorf("zeus: not started")
	}

	if z.isRunning() == true {
		z.stopClimate()
	}

	close(z.quit)
	<-z.done
	z.done = nil
	return nil
}

func (z *Zeus) hasZone(name string) bool {
	_, ok := z.definitions[name]
	return ok
}

func (z *Zeus) dispatcherForInterface(ifname string) (ArkeDispatcher, error) {
	d, ok := z.dispatchers[ifname]
	if ok == true {
		return d, nil
	}
	z.logger.Printf("Opening interface '%s'", ifname)
	intf, err := z.intfFactory(ifname)
	if err != nil {
		return nil, err
	}
	d = NewArkeDispatcher(ifname, intf)
	z.dispatchers[ifname] = d
	return d, nil
}

func (z *Zeus) checkSeason(season zeus.SeasonFile) error {
	for zoneName, _ := range season.Zones {
		if z.hasZone(zoneName) == false {
			return fmt.Errorf("missing zone '%s' %+v", zoneName, z.definitions)
		}
	}
	return nil
}

func (z *Zeus) setupZoneClimate(name, suffix string, definition ZoneDefinition, climate zeus.ZoneClimate) error {
	d, err := z.dispatcherForInterface(definition.CANInterface)
	if err != nil {
		return err
	}
	r, err := NewZoneClimateRunner(ZoneClimateRunnerOptions{
		Name:        name,
		FileSuffix:  suffix,
		Dispatcher:  d,
		Climate:     climate,
		OlympusHost: z.olympusHost,
		Definition:  definition,
	})
	if err != nil {
		return err
	}
	z.runners[name] = r
	return nil
}

func (z *Zeus) startClimate(season zeus.SeasonFile) (rerr error) {
	if z.isRunning() == true {
		return fmt.Errorf("Already started")
	}

	defer func() {
		if rerr == nil {
			return
		}
		z.closeDispatchers()
		z.reset()
	}()

	if err := z.checkSeason(season); err != nil {
		return fmt.Errorf("invalid season file: %s", err)
	}

	suffix := time.Now().Format("2006-01-02T150405")

	for name, climate := range season.Zones {
		err := z.setupZoneClimate(name, suffix, z.definitions[name], climate)
		if err != nil {
			return fmt.Errorf("Could not setup zone '%s': %s", name, err)
		}
	}

	z.logger.Printf("Starting climate")

	for _, d := range z.dispatchers {
		go d.Dispatch()
	}

	for _, r := range z.runners {
		go r.Run()
	}

	return nil
}

func (z *Zeus) closeRunners() {
	for name, r := range z.runners {
		err := r.Close()
		if err != nil {
			z.logger.Printf("runner for zone %s did not close gracefully: %s", name, err)
		}
	}
}

func (z *Zeus) closeDispatchers() {
	for name, d := range z.dispatchers {
		err := d.Close()
		if err != nil {
			z.logger.Printf("dispatcher for interface %s did not close gracefully: %s", name, err)
		}
	}
}

func (z *Zeus) reset() {
	z.runners = make(map[string]ZoneClimateRunner)
	z.dispatchers = make(map[string]ArkeDispatcher)
}

func (z *Zeus) stopClimate() error {
	if z.isRunning() == false {
		return fmt.Errorf("Not running")
	}

	z.logger.Printf("Stopping climate")

	z.closeRunners()
	z.closeDispatchers()
	z.reset()

	z.logger.Printf("Climate stopped")
	return nil
}

func (z *Zeus) StartClimate(season zeus.SeasonFile, unused *int) error {
	z.mx.Lock()
	defer z.mx.Unlock()

	return z.startClimate(season)
}

func (z *Zeus) StopClimate(ignored int, unused *int) error {
	z.mx.Lock()
	defer z.mx.Unlock()

	return z.stopClimate()
}

func (z *Zeus) isRunning() bool {
	return len(z.runners) != 0
}

func (z *Zeus) Running(ignored int, reply *bool) error {
	z.mx.Lock()
	defer z.mx.Unlock()

	*reply = z.isRunning()
	return nil
}
