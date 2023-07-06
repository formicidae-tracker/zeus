package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/formicidae-tracker/zeus/internal/zeus"
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

	fmt.Fprintf(res.File, "# Starting date %s\n%s\n", res.Start.Format(time.RFC3339Nano), header)

	return res, fname, nil
}

func readStartDate(r *bufio.Reader) (time.Time, error) {
	l, err := r.ReadString('\n')
	if err != nil {
		return time.Time{}, err
	}

	l = strings.TrimPrefix(l, "# Starting date")
	l = strings.TrimSpace(l)
	return time.Parse(time.RFC3339Nano, l)
}

func readNumAux(r *bufio.Reader) (int, error) {
	l, err := r.ReadString('\n')
	if err != nil {
		return 0, err
	}
	res := strings.Count(l, "(°C)") - 1
	if res < 0 {
		return 0, fmt.Errorf("invalid header '%s'", strings.TrimSpace(l))
	}
	return res, nil
}

func readClimateReport(r *bufio.Reader, start time.Time, numAux int) (zeus.ClimateReport, error) {
	res := zeus.ClimateReport{}
	l, err := r.ReadString('\n')
	if err != nil {
		return res, err
	}
	l = strings.TrimSpace(l)
	valuesStr := strings.Split(l, " ")
	if len(valuesStr) < 3+numAux {
		return res, fmt.Errorf("invalid line '%s': too few values", l)
	}
	ms, err := strconv.ParseInt(valuesStr[0], 10, 64)
	if err != nil {
		return res, fmt.Errorf("invalid timestamp '%s': %s", valuesStr[0], err)
	}
	res.Time = start.Add(time.Duration(ms) * time.Millisecond)
	h, err := strconv.ParseFloat(valuesStr[1], 64)
	if err != nil {
		return res, fmt.Errorf("invalid humidity '%s': %s", valuesStr[1], err)
	}
	res.Humidity = zeus.Humidity(h)
	res.Temperatures = make([]zeus.Temperature, numAux+1)
	for i, tStr := range valuesStr[2:(numAux + 3)] {
		t, err := strconv.ParseFloat(tStr, 64)
		if err != nil {
			return res, fmt.Errorf("invalid temperature '%s': %s", tStr, err)
		}
		res.Temperatures[i] = zeus.Temperature(t)
	}
	return res, nil
}

func ReadClimateFile(filename string) ([]zeus.ClimateReport, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	startDate, err := readStartDate(reader)
	if err != nil {
		return nil, err
	}
	numAux, err := readNumAux(reader)
	if err != nil {
		return nil, err
	}

	var res []zeus.ClimateReport
	for {
		cr, err := readClimateReport(reader, startDate, numAux)
		if err == io.EOF {
			break
		}
		if err != nil {
			return res, err
		}
		res = append(res, cr)
	}

	return res, nil
}
