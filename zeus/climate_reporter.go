package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/formicidae-tracker/zeus"
)

type ClimateReporter interface {
	Reporter
	ReportChannel() chan<- zeus.ClimateReport
}

type fileClimateReporter struct {
	File   *os.File
	NumAux int
	Format string
	Start  time.Time
	Chan   chan zeus.ClimateReport
}

func (n *fileClimateReporter) ReportChannel() chan<- zeus.ClimateReport {
	return n.Chan
}

func (n *fileClimateReporter) Report(ready chan<- struct{}) {
	close(ready)
	asInterface := make([]interface{}, n.NumAux+3)
	for cr := range n.Chan {
		if len(cr.Temperatures) != n.NumAux+1 {
			continue
		}
		asInterface[0] = cr.Time.Sub(n.Start).Nanoseconds() / 1e6
		asInterface[1] = cr.Humidity
		for i, t := range cr.Temperatures {
			asInterface[i+2] = t
		}

		fmt.Fprintf(n.File,
			n.Format,
			asInterface...)
	}
	n.File.Close()
}

func NewFileClimateReporter(filename string, numAux int) (ClimateReporter, string, error) {
	res := &fileClimateReporter{
		Chan:   make(chan zeus.ClimateReport, 10),
		Start:  time.Now(),
		NumAux: numAux,
	}

	var err error
	var fname string
	res.File, fname, err = zeus.CreateFileWithoutOverwrite(filename)
	if err != nil {
		return nil, "", err
	}

	res.Format = "%d %.2f %.2f" + strings.Repeat(" %.2f", numAux) + "\n"
	header := "# Time (ms) Relative Humidity (%) Temperature (°C)"
	for i := 0; i < numAux; i++ {
		header += fmt.Sprintf(" Aux %d (°C)", i+1)
	}

	fmt.Fprintf(res.File, "# Starting date %s\n%s\n", res.Start, header)

	return res, fname, nil
}
