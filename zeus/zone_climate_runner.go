package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/adrg/xdg"
	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/formicidae-tracker/zeus"
	"github.com/slack-go/slack"
)

type ZoneClimateRunner interface {
	Run()
	Close() error
}

type ZoneClimateRunnerOptions struct {
	Name        string
	Definition  ZoneDefinition
	FileSuffix  string
	Dispatcher  ArkeDispatcher
	Climate     zeus.ZoneClimate
	OlympusHost string
	SlackClient *slack.Client
	SlackUserID string
}

type zoneClimateRunner struct {
	logger     *log.Logger
	dispatcher ArkeDispatcher

	quit, done chan struct{}

	messages <-chan *StampedMessage

	interpoler      Interpoler
	capabilities    []capability
	presenceMonitor PresenceMonitorer
	alarmMonitor    AlarmMonitor

	reporters        []Reporter
	climateReporters []ClimateReporter
	stateReporters   []StateReporter
	alarmReporters   []AlarmReporter

	devices   map[arke.NodeClass]*Device
	callbacks map[arke.MessageClass][]callback
}

func (r *zoneClimateRunner) spawnAlarmMonitor(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		r.alarmMonitor.Monitor()
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		for event := range r.alarmMonitor.Outbound() {
			for _, reporter := range r.alarmReporters {
				reporter.AlarmChannel() <- event
			}

		}
		for _, reporter := range r.alarmReporters {
			close(reporter.AlarmChannel())
		}
		wg.Done()
	}()
}

func (r *zoneClimateRunner) spawnReporters(wg *sync.WaitGroup) {
	for _, reporter := range r.reporters {
		wg.Add(1)
		go func(reporter Reporter) {
			reporter.Report()
			wg.Done()
		}(reporter)
	}
}

func (r *zoneClimateRunner) spawnInterpoler(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		r.interpoler.Interpolate()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for newState := range r.interpoler.States() {
			for _, c := range r.capabilities {
				c.Action(newState)
			}
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for report := range r.interpoler.Reports() {
			for _, reporters := range r.stateReporters {
				reporters.StateChannel() <- report
			}
		}
		for _, reporters := range r.stateReporters {
			close(reporters.StateChannel())
		}
		wg.Done()
	}()

}

func (r *zoneClimateRunner) spawnPresenceMonitor(wg *sync.WaitGroup) {
	devices := []DeviceDefinition{}
	for _, device := range r.devices {
		devices = append(devices, DeviceDefinition{
			Class: device.Class,
			ID:    device.ID,
		})
	}
	wg.Add(1)
	go func() {
		r.presenceMonitor.Monitor(devices, r.alarmMonitor.Inbound())
		wg.Done()
	}()
}

func (r *zoneClimateRunner) spawnTasks(wg *sync.WaitGroup) {
	r.spawnAlarmMonitor(wg)
	r.spawnReporters(wg)
	r.spawnInterpoler(wg)
	r.spawnPresenceMonitor(wg)
}

func (r *zoneClimateRunner) stopTasks() {
	err := r.interpoler.Close()
	if err != nil {
		r.logger.Printf("interpoler did not close gracefully: %s", err)
	}
	err = r.presenceMonitor.Close()
	if err != nil {
		r.logger.Printf("presenceMonitorer did not close gracefully: %s", err)
	}

	for _, capability := range r.capabilities {
		capability.Close()
	}

	close(r.alarmMonitor.Inbound())
}

func (r *zoneClimateRunner) handleMessage(m *StampedMessage, wg *sync.WaitGroup) {
	switch m.M.MessageClassID() {
	case arke.HeartBeatMessage:
		r.presenceMonitor.Ping(m.M.(*arke.HeartBeatData).Class, m.ID)
	case arke.ErrorReportMessage:
		e := m.M.(*arke.ErrorReportData)
		r.alarmMonitor.Inbound() <- TimedAlarm{Alarm: zeus.NewDeviceInternalError(r.dispatcher.Name(), e.Class, e.ID, e.ErrorCode), Time: m.T}
	default:
		callbacks, ok := r.callbacks[m.M.MessageClassID()]
		if ok == false {
			return
		}
		wg.Add(1)
		go func(m *StampedMessage, alarms chan<- TimedAlarm) {
			for _, callback := range callbacks {
				err := callback(alarms, m)
				if err != nil {
					r.logger.Printf("callback error on %s: %s", m.M, err)
				}
			}
			wg.Done()
		}(m, r.alarmMonitor.Inbound())
	}
}

func (r *zoneClimateRunner) Run() {
	if r.quit != nil {
		return
	}
	r.quit = make(chan struct{})
	r.done = make(chan struct{})
	var wg, wgCallback sync.WaitGroup

	defer func() {
		wg.Wait()
		close(r.done)
	}()

	r.spawnTasks(&wg)
	r.logger.Printf("started")
	for {
		select {
		case <-r.quit:
			wgCallback.Wait()
			r.stopTasks()
			return
		case m := <-r.messages:
			r.handleMessage(m, &wgCallback)
		}
	}

}

func (r *zoneClimateRunner) Close() error {
	if r.quit == nil {
		return fmt.Errorf("already closed")
	}
	r.logger.Printf("closing")
	close(r.quit)
	<-r.done
	r.logger.Printf("closed")
	return nil
}

func (r *zoneClimateRunner) setUpInterpoler(o ZoneClimateRunnerOptions) error {
	interpoler, err := NewInterpoler(o.Name, o.Climate.States, o.Climate.Transitions)
	if err != nil {
		return err
	}
	r.interpoler = interpoler
	return nil
}

func (r *zoneClimateRunner) setUpRPC(o ZoneClimateRunnerOptions) error {
	if len(o.OlympusHost) == 0 {
		return nil
	}
	rpc, err := NewRPCReporter(o.Name, o.OlympusHost, o.Climate)
	if err != nil {
		return err
	}
	r.reporters = append(r.reporters, rpc)
	r.stateReporters = append(r.stateReporters, rpc)
	r.climateReporters = append(r.climateReporters, rpc)
	r.alarmReporters = append(r.alarmReporters, rpc)
	return nil
}

func (r *zoneClimateRunner) fileName(name, suffix, ftype string) (string, error) {
	return xdg.DataFile(filepath.Join("fort-experiments/climate", fmt.Sprintf("%s.%s.%s.txt", name, suffix, ftype)))
}

func (r *zoneClimateRunner) setUpFileReporters(o ZoneClimateRunnerOptions) error {
	climateFileName, err := r.fileName(o.Name, o.FileSuffix, "climate")
	if err != nil {
		return err
	}
	cr, _, err := NewFileClimateReporter(climateFileName)
	if err != nil {
		return err
	}
	r.reporters = append(r.reporters, cr)
	r.climateReporters = append(r.climateReporters, cr)

	alarmFileName, err := r.fileName(o.Name, o.FileSuffix, "alarms")
	if err != nil {
		return err
	}
	ar, err := NewFileAlarmReporter(alarmFileName)
	if err != nil {
		return err
	}
	r.reporters = append(r.reporters, ar)
	r.alarmReporters = append(r.alarmReporters, ar)
	return nil
}

func (r *zoneClimateRunner) setUpAlarmMonitor(o ZoneClimateRunnerOptions) error {
	alarmMonitor, err := NewAlarmMonitor(o.Name)
	if err != nil {
		return err
	}
	r.alarmMonitor = alarmMonitor
	return nil
}

func (r *zoneClimateRunner) setUpCapabilities(o ZoneClimateRunnerOptions) error {
	r.capabilities = ComputeClimateRequirements(&o.Climate, r.climateReporters)
	return nil
}

func (r *zoneClimateRunner) getDevice(d DeviceDefinition) *Device {
	dev, ok := r.devices[d.Class]
	if ok == true {
		return dev
	}
	dev = &Device{
		intf:  r.dispatcher.Interface(),
		Class: d.Class,
		ID:    d.ID,
	}
	r.devices[d.Class] = dev
	return dev
}

func (r *zoneClimateRunner) setUpDevices(o ZoneClimateRunnerOptions) error {
	for _, c := range r.capabilities {
		for _, class := range c.Requirements() {
			def := DeviceDefinition{
				Class: class,
				ID:    arke.NodeID(o.Definition.DevicesID),
			}
			r.getDevice(def)
		}
		c.SetDevices(r.devices)
		for mClass, callback := range c.Callbacks() {
			r.callbacks[mClass] = append(r.callbacks[mClass], callback)
		}
	}
	return nil
}

func (r *zoneClimateRunner) setUpSlackReporter(o ZoneClimateRunnerOptions) error {
	if o.SlackClient == nil || len(o.SlackUserID) == 0 {
		return nil
	}
	aReporter, err := NewSlackReporter(o.SlackClient, o.SlackUserID, o.Name)
	if err != nil {
		return err
	}
	r.reporters = append(r.reporters, aReporter)
	r.alarmReporters = append(r.alarmReporters, aReporter)
	return nil
}

func NewZoneClimateRunner(o ZoneClimateRunnerOptions) (r ZoneClimateRunner, err error) {
	res := &zoneClimateRunner{
		logger:          log.New(os.Stderr, "[zone/"+o.Name+"] ", 0),
		dispatcher:      o.Dispatcher,
		messages:        o.Dispatcher.Register(arke.NodeID(o.Definition.DevicesID)),
		presenceMonitor: NewPresenceMonitorer(o.Dispatcher.Name(), o.Dispatcher.Interface()),
		devices:         make(map[arke.NodeClass]*Device),
		callbacks:       make(map[arke.MessageClass][]callback),
	}

	setups := []func(ZoneClimateRunnerOptions) error{
		func(o ZoneClimateRunnerOptions) error { return res.setUpSlackReporter(o) },
		func(o ZoneClimateRunnerOptions) error { return res.setUpInterpoler(o) },
		func(o ZoneClimateRunnerOptions) error { return res.setUpAlarmMonitor(o) },
		func(o ZoneClimateRunnerOptions) error { return res.setUpRPC(o) },
		func(o ZoneClimateRunnerOptions) error { return res.setUpFileReporters(o) },
		func(o ZoneClimateRunnerOptions) error { return res.setUpCapabilities(o) },
		func(o ZoneClimateRunnerOptions) error { return res.setUpDevices(o) },
	}

	for _, s := range setups {
		if err := s(o); err != nil {
			return nil, err
		}
	}

	return res, nil

}
