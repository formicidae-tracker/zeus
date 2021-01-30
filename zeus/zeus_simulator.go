package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"sync"

	"github.com/formicidae-tracker/zeus"
)

type ZeusSimulator struct {
	stop, idle chan struct{}
	mx         sync.RWMutex
	zones      map[string]ZoneClimateRunner
	server     http.Server
}

type ZeusSimulatorArgs struct {
	hostname       string
	season         zeus.SeasonFile
	timeRatio      float64
	olympusAddress string
	rpcPort        int
}

func NewZeusSimulator(a ZeusSimulatorArgs) (s *ZeusSimulator, err error) {
	res := &ZeusSimulator{
		stop:  make(chan struct{}),
		idle:  make(chan struct{}),
		zones: make(map[string]ZoneClimateRunner),
	}
	address := fmt.Sprintf(":%d", a.rpcPort)
	l, err := res.setUpServer(address)
	if err != nil {
		return nil, err
	}
	go res.serve(l)

	defer func() {
		if err != nil {
			res.shutdown()
		}
	}()

	for zoneName, climate := range a.season.Zones {
		if err := func() error {
			res.mx.Lock()
			defer res.mx.Unlock()
			runner, err := NewZoneClimateStub(ZoneClimateStubArgs{
				hostname:       a.hostname,
				zoneName:       zoneName,
				climate:        climate,
				timeRatio:      a.timeRatio,
				olympusAddress: a.olympusAddress,
				rpcPort:        a.rpcPort,
			})
			if err != nil {
				return err
			}
			res.zones[zoneName] = runner
			go runner.Run()
			return nil
		}(); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (s *ZeusSimulator) shutdown() (err error) {
	s.mx.Lock()
	defer func() {
		s.mx.Unlock()
		if recover() != nil {
			err = errors.New("already closed")
		}
		<-s.idle
	}()
	for _, r := range s.zones {
		r.Close()
	}
	s.zones = make(map[string]ZoneClimateRunner)
	close(s.stop)

	return nil
}

func (s *ZeusSimulator) setUpServer(address string) (net.Listener, error) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	router := rpc.NewServer()
	router.RegisterName("Zeus", s)

	mux := http.NewServeMux()

	mux.Handle(rpc.DefaultRPCPath, router)
	s.server.Addr = address
	s.server.Handler = mux

	return l, nil
}

func (s *ZeusSimulator) serve(l net.Listener) {
	go func() {
		<-s.stop
		if err := s.server.Shutdown(context.Background()); err != nil {
			log.Printf("shutdown error: %s", err)
		}
		close(s.idle)
	}()

	err := s.server.Serve(l)
	if err == http.ErrServerClosed {
		return
	}
	if err != nil {
		log.Printf("server error: %s", err)
	}
}

func (s *ZeusSimulator) ClimateLog(args zeus.ZeusLogArgs, reply *zeus.ZeusClimateLogReply) error {
	s.mx.RLock()
	defer s.mx.RUnlock()
	var err error
	reply.Data, err = s.climateLog(args.ZoneName, args.Start, args.End)
	return err
}

func (s *ZeusSimulator) AlarmLog(args zeus.ZeusLogArgs, reply *zeus.ZeusAlarmLogReply) error {
	s.mx.RLock()
	defer s.mx.RUnlock()
	var err error
	reply.Data, err = s.alarmLog(args.ZoneName, args.Start, args.End)
	return err

}

func (s *ZeusSimulator) climateLog(zoneName string, start, end int) ([]zeus.ClimateReport, error) {
	r, ok := s.zones[zoneName]
	if ok == false {
		return nil, fmt.Errorf("unknown zone '%s'", zoneName)
	}
	return r.ClimateLog(start, end)
}

func (s *ZeusSimulator) alarmLog(zoneName string, start, end int) ([]zeus.AlarmEvent, error) {
	r, ok := s.zones[zoneName]
	if ok == false {
		return nil, fmt.Errorf("unknown zone '%s'", zoneName)
	}
	return r.AlarmLog(start, end)
}
