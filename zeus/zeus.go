package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adrg/xdg"
	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/zeus"
	"github.com/formicidae-tracker/zeus/zeuspb"
	"github.com/grandcat/zeroconf"
	"github.com/slack-go/slack"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Zeus struct {
	zeuspb.UnimplementedZeusServer
	intfFactory func(ifname string) (socketcan.RawInterface, error)

	logger      *log.Logger
	slackClient *slack.Client

	olympusHost string
	definitions map[string]ZoneDefinition

	dispatchers map[string]ArkeDispatcher
	runners     map[string]ZoneClimateRunner
	since       time.Time

	mx               sync.RWMutex
	quit, done, idle chan struct{}
}

func OpenZeus(c Config) (*Zeus, error) {
	if err := c.Check(); err != nil {
		return nil, fmt.Errorf("Invalid config: %s", err)
	}
	err := os.MkdirAll(filepath.Join(xdg.DataHome, "fort-experiments/climate"), 0755)
	if err != nil {
		return nil, err
	}
	z := &Zeus{
		intfFactory: socketcan.NewRawInterface,
		logger:      log.New(os.Stderr, "[zeus] ", 0),
		olympusHost: c.Olympus,
		definitions: c.Zones,
		runners:     make(map[string]ZoneClimateRunner),
		dispatchers: make(map[string]ArkeDispatcher),
	}
	if len(c.SlackToken) > 0 {
		z.logger.Printf("Slack notification are enabled")
		z.slackClient = slack.New(c.SlackToken)
	}

	z.restoreStaticState()

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
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", zeus.ZEUS_PORT))
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	zeuspb.RegisterZeusServer(server, z)

	go func() {
		<-z.quit
		server.GracefulStop()
		close(z.idle)
	}()

	if err := server.Serve(lis); err != nil {
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

func (z *Zeus) setupZoneClimate(name, suffix string, definition ZoneDefinition, climate zeus.ZoneClimate, userID string) error {
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
		SlackClient: z.slackClient,
		SlackUserID: userID,
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
	z.since = time.Now()
	suffix := z.since.Format("2006-01-02T150405")
	userID := ""

	if z.slackClient != nil && len(season.SlackUser) > 0 {
		var err error
		userID, err = FindSlackUser(z.slackClient, season.SlackUser)
		if err != nil {
			return err
		}
		z.logger.Printf("Will report to %s:%s", season.SlackUser, userID)
	}

	for name, climate := range season.Zones {
		err := z.setupZoneClimate(name, suffix, z.definitions[name], climate, userID)
		if err != nil {
			return fmt.Errorf("Could not setup zone '%s': %s", name, err)
		}
	}

	z.logger.Printf("Starting climate")

	for _, d := range z.dispatchers {
		ready := make(chan struct{})
		go d.Dispatch(ready)
		<-ready
	}

	for _, r := range z.runners {
		go r.Run()
	}

	z.saveStaticState(season)

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

	z.clearStaticState()

	z.logger.Printf("Stopping climate")

	z.closeRunners()
	z.closeDispatchers()
	z.reset()

	z.logger.Printf("Climate stopped")
	return nil
}

func (z *Zeus) StartClimate(c context.Context, request *zeuspb.StartRequest) (*zeuspb.Empty, error) {
	z.mx.Lock()
	defer z.mx.Unlock()

	compatible, err := zeus.VersionAreCompatible(zeus.ZEUS_VERSION, request.Version)
	if err != nil {
		return nil, err
	}

	if compatible == false {
		return nil, fmt.Errorf("client version (%s) is incompatible with service version (%s)", request.Version, zeus.ZEUS_VERSION)
	}

	seasonFile, err := zeus.ParseSeasonFile([]byte(request.SeasonFile))
	if err != nil {
		return nil, fmt.Errorf("could not read season file: %w", err)
	}
	err = z.startClimate(*seasonFile)
	if err != nil {
		return nil, err
	}
	return &zeuspb.Empty{}, nil
}

func (z *Zeus) StopClimate(c context.Context, e *zeuspb.Empty) (*zeuspb.Empty, error) {
	z.mx.Lock()
	defer z.mx.Unlock()

	if err := z.stopClimate(); err != nil {
		return nil, err
	}
	return &zeuspb.Empty{}, nil
}

func (z *Zeus) isRunning() bool {
	return len(z.runners) != 0
}

func (z *Zeus) GetStatus(c context.Context, e *zeuspb.Empty) (*zeuspb.Status, error) {
	z.mx.Lock()
	defer z.mx.Unlock()

	res := &zeuspb.Status{
		Running: z.isRunning(),
		Version: zeus.ZEUS_VERSION,
	}
	if res.Running == false {
		return res, nil
	}
	res.Since = timestamppb.New(z.since)
	for name, runner := range z.runners {
		status := runner.Last()
		if status == nil {
			continue
		}
		status.Name = name
		res.Zones = append(res.Zones, status)
	}
	return res, nil
}

func (z *Zeus) stateFilePath() (string, error) {
	return xdg.DataFile("fort-experiments/climate/current.season")
}

func (z *Zeus) saveStaticStateUnsafe(season zeus.SeasonFile) error {
	fpath, err := z.stateFilePath()
	if err != nil {
		return err
	}
	return season.WriteFile(fpath)
}

func (z *Zeus) saveStaticState(season zeus.SeasonFile) {
	if err := z.saveStaticStateUnsafe(season); err != nil {
		z.logger.Printf("could not save state: %s", err)
	}
}

func (z *Zeus) clearStaticStateUnsafe() error {
	filename, err := z.stateFilePath()
	if err != nil {
		return err
	}
	return os.RemoveAll(filename)
}

func (z *Zeus) clearStaticState() {
	if err := z.clearStaticStateUnsafe(); err != nil {
		z.logger.Printf("could not clear state: %s", err)
	}
}

func (z *Zeus) restoreStaticStateUnsafe() error {
	filename, err := z.stateFilePath()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			return
		}
		z.logger.Printf("clearing invalid state")
		z.clearStaticState()
	}()
	var season *zeus.SeasonFile = nil
	season, err = zeus.ReadSeasonFile(filename, bytes.NewBuffer(nil))
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			return nil
		}
		return err
	}
	err = z.startClimate(*season)
	return err
}

func (z *Zeus) restoreStaticState() {
	if err := z.restoreStaticStateUnsafe(); err != nil {
		z.logger.Printf("could not restore state: %s", err)
	}
}
