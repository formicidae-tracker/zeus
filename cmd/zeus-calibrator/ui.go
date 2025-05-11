package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"syscall"
	"time"

	tui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type logDisplay struct {
	lines      []string
	file       io.WriteCloser
	displayBox *widgets.Paragraph
}

type CalibratorUI struct {
	ctx    context.Context
	cancel context.CancelFunc
	logs   *logDisplay
	plot   *widgets.Plot

	needUpdate bool
}

var ui = newCalibratorUI()

func (d *logDisplay) Write(buf []byte) (int, error) {
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

	d.displayBox.Text = strings.Join(d.lines[max(0, len(d.lines)-availablesLines):], "\n")

	ui.MarkUpdate()
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
	res := &CalibratorUI{needUpdate: true}
	res.ctx, res.cancel = context.WithCancel(context.Background())
	return res
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

	ui.plot = widgets.NewPlot()

	grid := tui.NewGrid()
	tw, th := tui.TerminalDimensions()
	grid.SetRect(0, 0, tw, th)
	grid.Set(tui.NewRow(0.3, ui.plot), tui.NewRow(0.7, ui.logs.displayBox))

	// ticker to limit refresh FPS
	t := time.NewTicker(time.Second / 30)

	tui.Render(grid)
	for {
		select {
		case <-t.C:
			if ui.needUpdate == true {
				tui.Render(grid)
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
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.logs.resize()
				tui.Clear()
				ui.MarkUpdate()
			}
		}
	}

}
