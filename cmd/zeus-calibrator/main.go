package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/jessevdk/go-flags"
)

type Range struct {
	Low  float32
	High float32
}

func (r Range) Diff() float32 {
	return r.High - r.Low
}

func (r *Range) MarshalFlag() (string, error) {
	return fmt.Sprintf("%.2f-%.2f", r.Low, r.High), nil
}

func (r *Range) UnmarshalFlag(value string) error {
	parts := strings.Split(value, "-")
	if len(parts) > 2 {
		return fmt.Errorf("invalid range '%s': only one '-' is allowed", value)
	}
	v, err := strconv.ParseFloat(parts[0], 32)
	if err != nil {
		return fmt.Errorf("invalid range '%s': parsing '%s': %w", value, parts[0], err)
	}
	if len(parts) == 1 {
		r.Low = float32(v) - 5.0
		r.High = float32(v) + 5.0
	}
	r.Low = float32(v)
	v, err = strconv.ParseFloat(parts[1], 32)
	if err != nil {
		return fmt.Errorf("invalid range '%s': parsing '%s': %w", value, parts[1], err)
	}
	r.High = float32(v)
	if r.High <= r.Low {
		r.High, r.Low = r.Low, r.High
	}
	return nil
}

type Options struct {
	Interface   string `long:"interface" short:"i" description:"Interface to use" default:"slcan0"`
	ID          uint8  `long:"id" description:"ID to calibrate" default:"1"`
	Temperature Range  `long:"temperature"  description:"calibration temperature min" default:"28.0-32.0"`
	Humidity    Range  `long:"humidity"  description:"calibration humidity" default:"45.0-65.0"`
	Cycles      uint   `long:"cycles" description:"Number of cycle to determine system characteristics" default:"5"`
	DebugUI     bool   `long:"debug-ui" description:"debugs the UI"`
}

func (r Range) String() string {
	return fmt.Sprintf("%.02f-%.02f", r.Low, r.High)
}

func (o Options) check() error {
	if o.Temperature.High <= o.Temperature.Low {
		return fmt.Errorf("Invalid temperature range %s", o.Temperature)
	}

	if o.Humidity.High <= o.Humidity.Low {
		return fmt.Errorf("Invalid humidity range %s", o.Temperature)
	}
	return nil
}

func Execute() error {
	opts := Options{}

	if _, err := flags.Parse(&opts); err != nil {
		return err
	}

	if err := opts.check(); err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		ui.Loop()
		wg.Done()
	}()
	defer func() {
		ui.Close()
		wg.Wait()
	}()

	ui.temperatureRange = opts.Temperature
	ui.humidityRange = opts.Humidity

	if opts.DebugUI == true {
		return debugUI(opts)
	}

	c, err := newZeusCalibrator(opts)
	if err != nil {
		return err
	}
	defer c.Close()

	if err := c.calibrateTemperature(); err != nil {
		slog.Error("could not calibrate temperature", "error", err)
		return err
	}

	if err := c.calibrateHumidity(); err != nil {
		return err
	}

	slog.Info("got the following results",
		"temperature", c.temperatureParameters,
		"humidity", c.humidityParameters)

	slog.Info("press <q> to quit")

	<-c.ctx.Done()

	return nil
}

func debugUI(opts Options) error {
	ui.plotTimeWindow = time.Minute
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		cancel()
	}()

	ticker := time.NewTicker(500 * time.Millisecond)

	meanHumidity := (opts.Humidity.High + opts.Humidity.Low) / 2.0
	meanTemp := (opts.Temperature.High + opts.Temperature.Low) / 2.0

	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:
			ph := 2 * math.Pi * float64(t.Second()) / 10.0
			ui.PushZeusReport(t, &arke.ZeusReport{
				Humidity: float32(math.Sin(ph))*opts.Humidity.Diff() + meanHumidity,
				Temperature: [4]float32{
					float32(math.Cos(ph))*opts.Temperature.Diff() + meanTemp,
					float32(math.NaN()), float32(math.NaN()), float32(math.NaN())},
			})
		}
	}

}

func main() {

	if err := Execute(); err != nil {
		if errors.Is(err, context.Canceled) == true {
			return
		}

		slog.Error("unhandled error", "error", err)
		os.Exit(1)
	}
}
