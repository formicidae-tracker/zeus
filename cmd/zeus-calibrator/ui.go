package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/formicidae-tracker/zeus/cmd/zeus-calibrator/plot"
	tui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type logDisplay struct {
	mx         sync.Mutex
	lines      []string
	file       io.WriteCloser
	displayBox *widgets.Paragraph
}

type CalibratorUI struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	logs                  *logDisplay
	temperature, humidity *plot.Plot
	state                 *widgets.Paragraph

	temperatureRange, humidityRange Range
	plotTimeWindow                  time.Duration

	times   []time.Time
	reports []*arke.ZeusReport

	temperatureCommand, humidityCommand int16

	start time.Time

	needUpdate bool
}

var ui = newCalibratorUI()

func (d *logDisplay) Write(buf []byte) (int, error) {
	d.mx.Lock()
	defer d.mx.Unlock()
	newLines := strings.Split(string(buf), "\n")
	for _, line := range newLines {
		if len(line) == 0 {
			continue
		}
		d.lines = append(d.lines, line)
	}
	d.lines = d.lines[max(0, len(d.lines)-1000):]

	d.resize()

	return d.file.Write(buf)
}

func (d *logDisplay) resize() {
	availablesLines := d.displayBox.Dy()
	if d.displayBox.Border == true {
		availablesLines -= 2
	}

	if availablesLines > 0 {
		d.displayBox.Text = strings.Join(d.lines[max(0, len(d.lines)-availablesLines):], "\n")
	} else {
		d.displayBox.Text = ""
	}
	ui.MarkUpdate()
}

func (d *logDisplay) Resize() {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resize()
}

func newLogDisplay() (*logDisplay, error) {
	res := &logDisplay{
		lines:      make([]string, 0, 100),
		file:       nil,
		displayBox: widgets.NewParagraph(),
	}
	var err error
	res.file, err = os.Create(fmt.Sprintf("%s.%s.log", os.Args[0], time.Now().Format(time.RFC3339)))
	if err != nil {
		return nil, err
	}

	res.displayBox.Title = " Logs "
	res.displayBox.TitleStyle.Fg = tui.ColorCyan
	res.displayBox.BorderStyle.Fg = tui.ColorCyan

	return res, nil
}

func newCalibratorUI() *CalibratorUI {
	res := &CalibratorUI{needUpdate: true, plotTimeWindow: 5 * time.Minute}
	res.ctx, res.cancel = context.WithCancel(context.Background())
	return res
}

func (ui *CalibratorUI) PushCommands(temperature, humidity int16) {
	ui.temperatureCommand, ui.humidityCommand = temperature, humidity
	ui.updateState()
}

func (ui *CalibratorUI) updateState() {
	if len(ui.reports) == 0 {
		return
	}
	r := ui.reports[len(ui.reports)-1]
	ui.state.Text = fmt.Sprintf("Temperature: %.2f°C Humidity: %.2f%% R.H. Command: T=%d H=%d", r.Temperature[0], r.Humidity, ui.temperatureCommand, ui.humidityCommand)
	ui.MarkUpdate()
}
func (ui *CalibratorUI) PushZeusReport(t time.Time, r *arke.ZeusReport) {
	if len(ui.times) == 0 {
		ui.start = t
	}

	ui.updateState()

	minTime := t.Add(-1 * ui.plotTimeWindow)

	ui.times = append(ui.times, t)
	ui.reports = append(ui.reports, r)

	i := 0
	for ; i < len(ui.times); i += 1 {
		if ui.times[i].After(minTime) {
			break
		}
	}
	ui.times = ui.times[i:]
	ui.reports = ui.reports[i:]

	ui.updatePlots()
}

func (ui *CalibratorUI) updatePlots() {
	if len(ui.times) < 2 {
		return
	}

	times := make([]float64, 0, len(ui.times))
	temps := make([]float64, 0, len(ui.times))
	hums := make([]float64, 0, len(ui.times))

	for i, r := range ui.reports {
		times = append(times, ui.times[i].Sub(ui.start).Minutes())
		temps = append(temps, float64(r.Temperature[0]))
		hums = append(hums, float64(r.Humidity))
	}

	minX, maxX := times[0], times[0]+ui.plotTimeWindow.Minutes()

	ui.humidity.XData = times
	ui.humidity.YData = [][]float64{hums}
	ui.humidity.MinXVal, ui.humidity.MaxXVal = minX, maxX

	ui.temperature.XData = times
	ui.temperature.YData = [][]float64{temps}
	ui.temperature.MinXVal, ui.temperature.MaxXVal = minX, maxX

	ui.MarkUpdate()
}

func (ui *CalibratorUI) MarkUpdate() {
	ui.needUpdate = true
}

func (ui *CalibratorUI) Close() {
	ui.cancel()
}

func (ui *CalibratorUI) Loop() {
	if err := tui.Init(); err != nil {
		slog.Error("could not initialize termui", "error", err)
		os.Exit(2)
	}
	defer tui.Close()
	var err error
	ui.logs, err = newLogDisplay()
	if err != nil {
		slog.Error("could not initialize logs")
		tui.Close()
		os.Exit(2)
	}
	log.SetOutput(ui.logs)

	ui.temperature = plot.NewPlot()
	ui.temperature.Title = "Temperature (°C) / Time (min.)"
	ui.temperature.LineColors[0] = tui.ColorRed
	ui.temperature.MaxYVal = float64(ui.temperatureRange.High) + 1.0
	ui.temperature.MinYVal = float64(ui.temperatureRange.Low) - 1.0
	ui.humidity = plot.NewPlot()
	ui.humidity.Title = "Humidity (%R.H.) / Time (min.)"
	ui.humidity.LineColors[0] = tui.ColorCyan
	ui.humidity.MaxYVal = float64(ui.humidityRange.High) + 5.0
	ui.humidity.MinYVal = float64(ui.humidityRange.Low) - 5.0

	grid := tui.NewGrid()
	tw, th := tui.TerminalDimensions()
	grid.SetRect(0, 3, tw, th)
	grid.Set(
		tui.NewRow(0.4,
			tui.NewCol(0.5, ui.temperature),
			tui.NewCol(0.5, ui.humidity),
		),
		tui.NewRow(0.6, ui.logs.displayBox),
	)

	ui.state = widgets.NewParagraph()
	ui.state.SetRect(0, 0, tw, 3)
	ui.state.Title = " Box State "

	// ticker to limit refresh FPS
	t := time.NewTicker(time.Second / 30)

	tui.Render(ui.state, grid)
	for {
		select {
		case <-t.C:
			if ui.needUpdate == true {
				tui.Render(ui.state, grid)
				ui.needUpdate = false
			}
		case <-ui.ctx.Done():
			return
		case e := <-tui.PollEvents():
			switch e.ID {
			case "q", "<C-c>":
				syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			case "<Resize":
				payload := e.Payload.(tui.Resize)
				grid.SetRect(0, 3, payload.Width, payload.Height)
				ui.state.SetRect(0, 0, payload.Width, 3)
				ui.logs.Resize()
				tui.Clear()
				ui.updatePlots()
				ui.MarkUpdate()
			}
		}
	}

}
