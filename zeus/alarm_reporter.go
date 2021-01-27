package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/formicidae-tracker/zeus"
)

type AlarmReporter interface {
	Reporter
	AlarmChannel() chan<- zeus.AlarmEvent
}

type fileAlarmReporter struct {
	file   *os.File
	events chan zeus.AlarmEvent
}

func (r *fileAlarmReporter) Report(ready chan<- struct{}) {
	close(ready)
	for event := range r.events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		fmt.Fprintf(r.file, "%s\n", data)
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
		file:   file,
		events: make(chan zeus.AlarmEvent),
	}, nil
}

func ReadAlarmLogFile(filename string) ([]zeus.AlarmEvent, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var res []zeus.AlarmEvent
	reader := bufio.NewReader(f)
	for {
		l, err := reader.ReadString('\n')
		if err == io.EOF {
			return res, nil
		}
		if err != nil {
			return res, err
		}
		event := zeus.AlarmEvent{}
		err = json.Unmarshal([]byte(l), &event)
		if err != nil {
			return res, err
		}
		res = append(res, event)
	}
}
