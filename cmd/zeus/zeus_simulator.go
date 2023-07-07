package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/formicidae-tracker/olympus/pkg/tm"
	"github.com/formicidae-tracker/zeus/internal/zeus"
	"github.com/formicidae-tracker/zeus/pkg/zeuspb"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

type ZeusSimulator struct {
	zeuspb.UnimplementedZeusServer
	stop, idle chan struct{}
	mx         sync.RWMutex
	zones      map[string]ZoneClimateRunner
	server     *grpc.Server
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

	options := []grpc.ServerOption{}
	if tm.Enabled() {
		options = append(options,
			grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
			grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
		)
	}

	s.server = grpc.NewServer(options...)
	zeuspb.RegisterZeusServer(s.server, s)

	return l, nil
}

func (s *ZeusSimulator) serve(l net.Listener) {
	go func() {
		<-s.stop
		s.server.GracefulStop()
		close(s.idle)
	}()

	err := s.server.Serve(l)
	if err != nil {
		log.Printf("server error: %s", err)
	}
}
