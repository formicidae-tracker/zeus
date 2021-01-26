package main

import (
	"log"
	"os"

	"github.com/formicidae-tracker/zeus"
)

type AlarmReporter interface {
	Reporter
	AlarmChannel() chan<- zeus.AlarmEvent
}

type fileAlarmReporter struct {
	logger *log.Logger
	file   *os.File
	events chan zeus.AlarmEvent
}

func (r *fileAlarmReporter) Report(ready chan<- struct{}) {
	close(ready)
	for event := range r.events {
		r.logger.Printf("%+v", event)

	}
	r.file.Close()
}

func (r *fileAlarmReporter) AlarmChannel() chan<- zeus.AlarmEvent {
	return r.events
}

func NewFileAlarmReporter(filename string) (AlarmReporter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return &fileAlarmReporter{
		logger: log.New(file, "", log.LstdFlags),
		file:   file,
		events: make(chan zeus.AlarmEvent),
	}, nil
}
