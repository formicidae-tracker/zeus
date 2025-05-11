package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/jessevdk/go-flags"
)

type Range struct {
	Low  float32
	High float32
}

func (r *Range) MarshalFlag() (string, error) {
	return fmt.Sprintf("%.2f-%.2f", r.Low, r.High), nil
}

func (r *Range) UnmarhsalFlag(value string) error {
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
}

type timedMessage struct {
	T  time.Time
	ID arke.NodeID
	M  arke.ReceivableMessage
}

type canInterface struct {
	Messages chan timedMessage
	intf     socketcan.RawInterface
	logger   *slog.Logger
}

func (c *canInterface) listen() {
	defer close(c.Messages)
	for {
		f, err := c.intf.Receive()
		now := time.Now()
		if err != nil {
			c.logger.Error("could not read frame", "error", err)
			return
		}
		m, id, err := arke.ParseMessage(&f)
		if err != nil {
			c.logger.Error("could not parse frame", "error", err)
			continue
		}

		c.Messages <- timedMessage{T: now, M: m, ID: id}
	}
}

func OpenCanInterface(name string) (*canInterface, error) {
	intf, err := socketcan.NewRawInterface(name)
	if err != nil {
		return nil, err
	}

	res := &canInterface{
		Messages: make(chan timedMessage, 100),
		intf:     intf,
		logger:   slog.With("canInterface", name),
	}

	go res.listen()
	return res, nil
}

func (intf *canInterface) Close() error {
	intf.logger.Info("closing")
	return intf.Close()
}

func (intf *canInterface) Send(c socketcan.CanFrame) error {
	intf.logger.Debug("sending frame", "frame", c)
	return intf.intf.Send(c)
}

func Execute() error {
	opts := Options{}

	if _, err := flags.Parse(&opts); err != nil {
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

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		cancel()
	}()

	intf, err := OpenCanInterface(opts.Interface)
	if err != nil {
		return err
	}
	defer intf.Close()

	ID := arke.NodeID(opts.ID)

	if err := pingZeus(ctx, intf, ID); err != nil {
		return fmt.Errorf("could not ping zeus %d: %w", opts.ID, err)
	}

	calibrateTemperature(ctx, intf, opts)

	return nil
}

func pingZeus(ctx context.Context, intf *canInterface, ID arke.NodeID) error {
	if err := intf.Send(arke.MakePing(arke.ZeusClass)); err != nil {
		return fmt.Errorf("could not send ping: %w", err)
	}

	deadline := time.After(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-deadline:
			return fmt.Errorf("ping timeouted")
		case tm, ok := <-intf.Messages:
			if ok == false {
				return fmt.Errorf("ping on closed interface")
			}

			if tm.M.MessageClassID() != arke.HeartBeatMessage {
				break
			}

			hd := tm.M.(*arke.HeartBeatData)
			if hd.Class != arke.ZeusClass || tm.ID != ID {
				break
			}
			return nil
		}
	}

}

func waitZeusReport(ctx context.Context,
	timeout time.Duration,
	intf *canInterface,
	ID arke.NodeID,
	cond func(*arke.ZeusReport) bool) (time.Time, *arke.ZeusReport, error) {

	wctx, _ := context.WithTimeoutCause(ctx, timeout, fmt.Errorf("could not reach target in %s", timeout))

	for {
		select {
		case tm := <-intf.Messages:
			if tm.ID != ID || tm.M.MessageClassID() != arke.ZeusReportMessage {
				continue
			}
			m := tm.M.(*arke.ZeusReport)
			if cond(m) == true {
				return tm.T, m, nil
			}
		case <-wctx.Done():
			if wctx.Err() != context.Canceled {
				return time.Time{}, nil, context.Cause(wctx)
			}
			return time.Time{}, nil, nil
		}
	}
}

func calibrateTemperature(ctx context.Context, intf *canInterface, opt Options) error {

	defer intf.Send(MakeZeusControlPoint(arke.NodeID(opt.ID), 1024, 1024))

	slog.Info("ramping up temperature", "T", opt.Temperature.High)

	bias := 0
	swing := 511
	var ellapsedUp, ellapsedDown time.Duration
	Ku := func() float32 {
		outAmp := opt.Temperature.High - opt.Temperature.Low
		return 4.0 * float32(swing) * 2.0 / (math.Pi * outAmp * 0.5)
	}

	Tu := func() float32 {
		return float32((ellapsedUp + ellapsedDown).Seconds())
	}

	timeLow := time.Now()
	if err := intf.Send(MakeZeusControlPoint(arke.NodeID(opt.ID),
		int16(clamp(bias+swing, -511, 511)),
		0)); err != nil {
		return err
	}

	timeHigh, _, err := waitZeusReport(
		ctx,
		10*time.Minute,
		intf,
		arke.NodeID(opt.ID),
		func(r *arke.ZeusReport) bool {
			return r.Temperature[0] >= opt.Temperature.High
		})
	if err != nil {
		return err
	}
	ellapsedUp = timeHigh.Sub(timeLow)
	slog.Info("temperature reached", "T", opt.Temperature.High, "ellapsed", ellapsedUp)

	onTarget := 0

	for i := 0; i < int(opt.Cycles); i += 1 {
		log := slog.With("cycle", i)
		log.Info("cooling down", "T", opt.Temperature.Low, "bias", bias, "amplitude", swing)

		if err := intf.Send(MakeZeusControlPoint(arke.NodeID(opt.ID),
			int16(clamp(bias-swing, -511, 511)),
			0)); err != nil {
			return err
		}

		timeLow, r, err := waitZeusReport(ctx, 5*time.Minute,
			intf, arke.NodeID(opt.ID),
			func(r *arke.ZeusReport) bool {
				return r.Temperature[0] <= opt.Temperature.Low
			})
		if err != nil {
			return err
		}
		ellapsedDown = timeLow.Sub(timeHigh)

		log.Info("temperature reached", "T", r.Temperature[0], "ellapsed", ellapsedDown)
		log.Info("heating up", "T", opt.Temperature.High, "bias", bias, "amplitude", swing)
		if err := intf.Send(MakeZeusControlPoint(arke.NodeID(opt.ID), int16(clamp(bias+swing, -511, 511)), 0)); err != nil {
			return err
		}

		timeHigh, r, err := waitZeusReport(ctx, 5*time.Minute,
			intf, arke.NodeID(opt.ID),
			func(r *arke.ZeusReport) bool {
				return r.Temperature[0] >= opt.Temperature.High
			})
		if err != nil {
			return err
		}

		ellapsedUp = timeHigh.Sub(timeLow)

		log.Info("temperature reached", "T", r.Temperature[0], "ellapsed", ellapsedUp)

		relativeDifference := math.Abs(ellapsedUp.Seconds()-ellapsedDown.Seconds()) / (ellapsedUp.Seconds() + ellapsedDown.Seconds())

		bias += clamp(int(float64(swing)*relativeDifference), -255, 255)
		if bias >= 0 {
			swing = 511 - bias
		} else {
			swing = 511 + bias
		}

		log.Info("cycle result",
			"duration", ellapsedUp+ellapsedDown,
			"difference", relativeDifference,
			"Ku", Ku(),
			"Tu", Tu())

		if relativeDifference < 0.05 {
			onTarget += 1
			if onTarget >= 2 {
				log.Info("twice on target: DONE")
				break
			}
		} else {
			onTarget = 0
		}

	}
	Kp := Ku() / 3.0
	Ki := 2.0 * Kp / Tu()
	Kd := Tu() / 3.0 * Ku()
	slog.Info("got following PID parameters",
		"Ku", Ku(),
		"Tu", Tu(),
		"Kp", Kp,
		"Ki", Ki,
		"Kd", Kd)

	return nil
}

func main() {

	if err := Execute(); err != nil {
		slog.Error("unhandled error", "error", err)
		os.Exit(1)
	}
}
