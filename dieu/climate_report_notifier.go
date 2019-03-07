package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"git.tuleu.science/fort/dieu"
)

type ClimateReportNotifier interface {
	C() chan<- dieu.ClimateReport
	Notify()
}

type fileClimateReportNotifier struct {
	File  *os.File
	Start time.Time
	Chan  chan dieu.ClimateReport
}

func (n *fileClimateReportNotifier) C() chan<- dieu.ClimateReport {
	return n.Chan
}

func (n *fileClimateReportNotifier) Notify() {
	for cr := range n.Chan {
		fmt.Fprintf(n.File,
			"%d %.2f %.2f %.2f %.2f %.2f\n",
			cr.Time.Sub(n.Start).Nanoseconds()/1e6,
			cr.Humidity,
			cr.Temperatures[0],
			cr.Temperatures[1],
			cr.Temperatures[2],
			cr.Temperatures[3])
	}
	n.File.Close()
}

func NewFileClimateReportNotifier(filename string) (ClimateReportNotifier, error) {
	res := &fileClimateReportNotifier{
		Chan:  make(chan dieu.ClimateReport, 10),
		Start: time.Now(),
	}

	var err error
	var fname string
	res.File, fname, err = dieu.CreateFileWithoutOverwrite(filename)
	if err != nil {
		return nil, err
	}
	log.Printf("Will save climate data in '%s'", fname)
	fmt.Fprintf(res.File, "#Starting date %s\n#Time(ms) Relative Humidity (%%) Temperature (°C) Temperature (°C) Temperature (°C) Temperature (°C)\n", res.Start)

	return res, nil
}
