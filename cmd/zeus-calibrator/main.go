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
	Interface                         string       `long:"interface" short:"i" description:"Interface to use" default:"slcan0"`
	ID                                uint8        `long:"id" description:"ID to calibrate" default:"1"`
	Temperature                       Range        `long:"temperature"  description:"calibration temperature min" default:"28.0-32.0"`
	Humidity                          Range        `long:"humidity"  description:"calibration humidity" default:"45.0-65.0"`
	Cycles                            uint         `long:"cycles" description:"Number of cycle to determine system characteristics" default:"5"`
	DebugUI                           bool         `long:"debug-ui" description:"debugs the UI"`
	TemperaturePID                    PIDParameter `group:"temperature-pid" namespace:"temperature"`
	HumidityPID                       PIDParameter `group:"humidity-pid" namespace:"humidity"`
	temperaturePIDSet, humidityPIDSet bool
	TestCycle                         time.Duration `long:"cycle-duration" default:"30m"`
}

func (r Range) String() string {
	return fmt.Sprintf("%.02f-%.02f", r.Low, r.High)
}

func (o *Options) check() error {
	if o.Temperature.High <= o.Temperature.Low {
		return fmt.Errorf("Invalid temperature range %s", o.Temperature)
	}

	if o.Humidity.High <= o.Humidity.Low {
		return fmt.Errorf("Invalid humidity range %s", o.Temperature)
	}

	var err error
	o.temperaturePIDSet, err = o.TemperaturePID.Check()
	if err != nil {
		return err
	}
	o.humidityPIDSet, err = o.HumidityPID.Check()
	if err != nil {
		return err
	}

	return nil
}

func makePIDConfig(p PIDParameter, divPower uint8, intDivPower uint8) (arke.PDConfig, error) {

	div := float32(math.Pow(2, float64(divPower)))
	intDiv := float32(math.Pow(2, float64(intDivPower)))
	maxKx := 255.0 / div
	maxKi := 255.0 / intDiv

	if p.Kp >= maxKx {
		return arke.PDConfig{}, fmt.Errorf("Kp=%.2f is out of range (max allowed is %.2f for div=%d", p.Kp, maxKx, divPower)
	}

	if p.Kd >= maxKx {
		return arke.PDConfig{}, fmt.Errorf("Kd=%.2f is out of range (max allowed is %.2f for div=%d", p.Kd, maxKx, divPower)
	}

	if p.Ki >= maxKi {
		return arke.PDConfig{}, fmt.Errorf("Ki=%.5f is out of range (max allowed is %.5f for div=%d", p.Ki, maxKi, intDivPower)
	}

	return arke.PDConfig{
		ProportionnalMultiplier: uint8(p.Kp * div),
		DerivativeMultiplier:    uint8(p.Kd * div),
		IntegralMultiplier:      uint8(p.Ki * intDiv),
		DividerPower:            divPower,
		DividerPowerIntegral:    intDivPower,
	}, nil

}

type LogableZeusConfig arke.ZeusConfig

func logPDConfig(c arke.PDConfig) slog.Value {
	div := math.Pow(2, float64(c.DividerPower))
	intDiv := math.Pow(2, float64(c.DividerPowerIntegral))
	return slog.GroupValue(
		slog.Int("KpMult", int(c.ProportionnalMultiplier)),
		slog.Int("KpDiv", int(div)),
		slog.Int("KdMult", int(c.DerivativeMultiplier)),
		slog.Int("KdDiv", int(div)),
		slog.Int("KiMult", int(c.IntegralMultiplier)),
		slog.Int("KiDiv", int(intDiv)),
		slog.Float64("Kp", float64(c.ProportionnalMultiplier)/div),
		slog.Float64("Kd", float64(c.DerivativeMultiplier)/div),
		slog.Float64("Ki", float64(c.IntegralMultiplier)/intDiv),
	)
}

func (c LogableZeusConfig) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("humidity", logPDConfig(c.Humidity)),
		slog.Any("temperature", logPDConfig(c.Temperature)),
	)
}

func testPID(opts Options) error {

	config := arke.ZeusConfig{}
	var err error
	config.Temperature, err = makePIDConfig(opts.TemperaturePID, 2, 10)
	if err != nil {
		return err
	}

	config.Humidity, err = makePIDConfig(opts.HumidityPID, 4, 12)
	if err != nil {
		return err
	}

	intf, err := OpenCanInterface(opts.Interface)
	if err != nil {
		return err
	}
	defer intf.Close()

	go func() {
		for tm := range intf.Messages {
			if tm.ID != arke.NodeID(opts.ID) {
				continue
			}
			cls := tm.M.MessageClassID()
			switch cls {
			case arke.ZeusControlPointMessage:
				cp := tm.M.(*arke.ZeusControlPoint)
				ui.PushCommands(cp.Temperature, cp.Humidity)
			case arke.ZeusReportMessage:
				r := tm.M.(*arke.ZeusReport)
				ui.PushZeusReport(tm.T, r)
			default:
				break
			}
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sigint := make(chan os.Signal, 2)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		cancel()
	}()

	slog.Info("sending config", "config", LogableZeusConfig(config))
	if err = intf.Send(buildSendableMessage(arke.NodeID(opts.ID), &config)); err != nil {
		return err
	}

	t := time.NewTicker(opts.TestCycle)
	targetHigh := false

	switchTarget := func() error {
		targetHigh = !targetHigh
		sp := &arke.ZeusSetPoint{
			Humidity:    opts.Humidity.Low,
			Temperature: opts.Temperature.Low,
			Wind:        255,
		}
		if targetHigh == true {
			sp.Humidity = opts.Humidity.High
			sp.Temperature = opts.Temperature.High
		}
		slog.Info("switching set point", "setPoint", sp, "next", opts.TestCycle)
		return intf.Send(buildSendableMessage(arke.NodeID(opts.ID), sp))
	}

	if err = switchTarget(); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := switchTarget(); err != nil {
				return err
			}
		}
	}
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
	ui.plotTimeWindow = 10 * time.Minute

	ui.WaitReady()

	if opts.DebugUI == true {
		return debugUI(opts)
	}

	if opts.temperaturePIDSet && opts.humidityPIDSet {
		return testPID(opts)
	}

	c, err := newZeusCalibrator(opts)
	if err != nil {
		return err
	}
	defer c.Close()

	if opts.temperaturePIDSet == false {
		if err := c.calibrateTemperature(); err != nil {
			slog.Error("could not calibrate temperature", "error", err)
			return err
		}
	} else {
		c.temperatureParameters = opts.TemperaturePID
	}

	if opts.humidityPIDSet == true {
		return nil
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
				Humidity: float32(0.5*math.Sin(ph))*opts.Humidity.Diff() + meanHumidity,
				Temperature: [4]float32{
					float32(0.5*math.Cos(ph))*opts.Temperature.Diff() + meanTemp,
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
