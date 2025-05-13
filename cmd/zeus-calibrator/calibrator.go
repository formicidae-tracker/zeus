package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"time"

	"github.com/formicidae-tracker/libarke/src-go/arke"
)

type PIDParameter struct {
	Kp, Kd, Ki float32
}

const (
	TEMP_RAMP_UP     = 25 * time.Minute
	TEMP_CHANGE      = 20 * time.Minute
	HUMIDITY_RAMP_UP = 8 * time.Minute
	HUMIDITY_CHANGE  = 3 * time.Minute
)

func pidParametersFromKuTu(Ku, Tu float32) PIDParameter {
	kp := Ku / 3.0
	ki := 2.0 / 3.0 * Ku / Tu
	kd := Tu / 9.0 * Ku
	return PIDParameter{kp, kd, ki}
}

type timedReport struct {
	T time.Time
	R *arke.ZeusReport
}

type zeusCalibrator struct {
	intf *canInterface
	ID   arke.NodeID
	ctx  context.Context

	reports chan timedReport

	temperatureTargets Range
	humidityTargets    Range
	cycles             int

	temperatureParameters PIDParameter
	humidityParameters    PIDParameter

	humidityCommand, temperatureCommand int16

	temperatureMeasuredRange, humidityMeasuredRange Range

	onNewReport func(tr timedReport)

	bias, amplitude int
}

func (c *zeusCalibrator) sendCommands() error {
	ui.PushCommands(c.temperatureCommand, c.humidityCommand)

	return c.intf.Send(MakeZeusControlPoint(c.ID, c.temperatureCommand, c.humidityCommand))
}

func (c *zeusCalibrator) resetMeasuredRange() {
	c.temperatureMeasuredRange = Range{Low: 100.0, High: 0.0}
	c.humidityMeasuredRange = Range{Low: 100.0, High: 0.0}
}

func (c *zeusCalibrator) updateMeasureRange(r *arke.ZeusReport) {
	c.temperatureMeasuredRange.Low = min(c.temperatureMeasuredRange.Low, r.Temperature[0])
	c.temperatureMeasuredRange.High = max(c.temperatureMeasuredRange.High, r.Temperature[0])

	c.humidityMeasuredRange.Low = min(c.humidityMeasuredRange.Low, r.Humidity)
	c.humidityMeasuredRange.High = max(c.humidityMeasuredRange.High, r.Humidity)

}

func newZeusCalibrator(opts Options) (*zeusCalibrator, error) {
	res := &zeusCalibrator{
		ID: arke.NodeID(opts.ID),

		reports:            make(chan timedReport, 100),
		temperatureTargets: opts.Temperature,
		humidityTargets:    opts.Humidity,
		cycles:             int(opts.Cycles),
		onNewReport:        func(tr timedReport) {},
	}

	var cancel context.CancelFunc
	res.ctx, cancel = context.WithCancel(context.Background())

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		cancel()
	}()
	var err error
	res.intf, err = OpenCanInterface(opts.Interface)
	if err != nil {
		return nil, err
	}

	if err := res.pingZeus(); err != nil {
		res.intf.Close()
		return nil, err
	}

	go res.interceptReports()

	return res, nil
}

func (c *zeusCalibrator) pingZeus() error {
	if err := c.intf.Send(arke.MakePing(arke.ZeusClass)); err != nil {
		return fmt.Errorf("could not send ping: %w", err)
	}

	deadline := time.After(500 * time.Millisecond)
	for {
		select {
		case <-c.ctx.Done():
			return nil
		case <-deadline:
			return fmt.Errorf("ping timeouted")
		case tm, ok := <-c.intf.Messages:
			if ok == false {
				return fmt.Errorf("ping on closed interface")
			}

			if tm.M.MessageClassID() != arke.HeartBeatMessage {
				break
			}

			hd := tm.M.(*arke.HeartBeatData)
			if hd.Class != arke.ZeusClass || tm.ID != c.ID {
				break
			}
			return nil
		}
	}

}

func (c *zeusCalibrator) interceptReports() {
	defer close(c.reports)
	for {
		select {
		case <-c.ctx.Done():
			return
		case tm, ok := <-c.intf.Messages:
			if ok == false {
				return
			}
			if tm.ID != c.ID || tm.M.MessageClassID() != arke.ZeusReportMessage {
				break
			}
			r := tm.M.(*arke.ZeusReport)
			c.updateMeasureRange(r)

			ui.PushZeusReport(tm.T, r)
			c.reports <- timedReport{T: tm.T, R: r}
		}

	}

}

func (c *zeusCalibrator) Close() error {
	slog.Info("closing calibrator: send reset control point")
	c.intf.Send(MakeZeusControlPoint(c.ID, 1024, 1024))
	return c.intf.Close()
}

func (c *zeusCalibrator) waitZeusReport(timeout time.Duration,
	cond func(*arke.ZeusReport) bool) (time.Time, *arke.ZeusReport, error) {

	wctx, cancel := context.WithTimeoutCause(c.ctx, timeout, fmt.Errorf("could not reach target in %s", timeout))
	defer cancel()

	for {
		select {
		case tr, ok := <-c.reports:
			if ok == false {
				return time.Time{}, nil, fmt.Errorf("channel closed")
			}
			c.onNewReport(tr)
			if cond(tr.R) == true {
				return tr.T, tr.R, nil
			}
		case <-wctx.Done():
			return time.Time{}, nil, context.Cause(wctx)
		}
	}
}

func abs[T ~int | ~float32 | ~float64](v T) T {
	if v < 0 {
		return -v
	}
	return v
}

func (c *zeusCalibrator) Ku(maxV, minV float32) float32 {

	return 4.0 * float32(c.amplitude) / (math.Pi * abs(maxV-minV))
}

func (c *zeusCalibrator) bangBangTarget(high bool) int16 {
	if high == true {
		return int16(clamp(c.bias+c.amplitude, -511, 511))
	}
	return int16(clamp(c.bias-c.amplitude, -511, 511))
}

func (c *zeusCalibrator) calibrateTemperature() error {

	defer c.intf.Send(MakeZeusControlPoint(c.ID, 1024, 1024))

	c.temperatureCommand, c.humidityCommand = 0, 0
	c.onNewReport = func(timedReport) {
		c.sendCommands()
	}
	defer func() { c.onNewReport = func(timedReport) {} }()

	slog.Info("ramping up temperature", "T", c.temperatureTargets.High)
	c.bias = 0
	c.amplitude = 511
	var ellapsedUp, ellapsedDown time.Duration

	Tu := func() float32 {
		return float32((ellapsedUp + ellapsedDown).Seconds())
	}

	c.temperatureCommand, c.humidityCommand = c.bangBangTarget(true), 0
	timeLow := time.Now()
	if err := c.sendCommands(); err != nil {
		return err
	}

	timeHigh, _, err := c.waitZeusReport(
		TEMP_RAMP_UP,
		func(r *arke.ZeusReport) bool {
			return r.Temperature[0] >= c.temperatureTargets.High
		})
	if err != nil {
		return err
	}
	ellapsedUp = timeHigh.Sub(timeLow)
	slog.Info("temperature reached", "T", c.temperatureTargets.High, "ellapsed", ellapsedUp)

	onTarget := 0

	for i := 0; i < c.cycles; i += 1 {
		c.resetMeasuredRange()
		log := slog.With("cycle", i)
		log.Info("cooling down", "T", c.temperatureTargets.Low, "bias", c.bias, "amplitude", c.amplitude)

		c.temperatureCommand, c.humidityCommand = c.bangBangTarget(false), -c.bangBangTarget(false)/2
		if err := c.sendCommands(); err != nil {
			return err
		}

		var r *arke.ZeusReport
		timeLow, r, err = c.waitZeusReport(
			TEMP_CHANGE,
			func(r *arke.ZeusReport) bool {
				return r.Temperature[0] <= c.temperatureTargets.Low
			})
		if err != nil {
			return err
		}
		ellapsedDown = timeLow.Sub(timeHigh)

		log.Info("temperature reached", "T", r.Temperature[0], "ellapsed", ellapsedDown)
		log.Info("heating up", "T", c.temperatureTargets.High, "bias", c.bias, "amplitude", c.amplitude)

		c.temperatureCommand, c.humidityCommand = c.bangBangTarget(true), 0
		if err := c.sendCommands(); err != nil {
			return err
		}

		timeHigh, r, err = c.waitZeusReport(
			TEMP_CHANGE,
			func(r *arke.ZeusReport) bool {
				return r.Temperature[0] >= c.temperatureTargets.High
			})
		if err != nil {
			return err
		}

		ellapsedUp = timeHigh.Sub(timeLow)

		log.Info("temperature reached", "T", r.Temperature[0], "ellapsed", ellapsedUp)

		relativeDifference := c.updateBangBang(ellapsedUp, ellapsedDown)

		log.Info("cycle result",
			"duration", ellapsedUp+ellapsedDown,
			"difference", relativeDifference,
			"Ku", c.Ku(c.temperatureMeasuredRange.Low, c.temperatureMeasuredRange.High),
			"Tu", Tu())

		if math.Abs(relativeDifference) < 0.05 {
			onTarget += 1
			if onTarget >= 2 {
				log.Info("twice on target: DONE")
				break
			}
		} else {
			onTarget = 0
		}

	}
	ku := c.Ku(c.temperatureTargets.Low, c.temperatureMeasuredRange.High)
	c.temperatureParameters = pidParametersFromKuTu(ku, Tu())
	slog.Info("got following Temperature PID parameters",
		"Ku", ku,
		"Tu", Tu(),
		"PID", c.temperatureParameters,
	)

	return nil
}

func (c *zeusCalibrator) updateBangBang(ellapsedUp, ellapsedDown time.Duration) float64 {
	relativeDifference := (ellapsedUp - ellapsedDown).Seconds() / (ellapsedUp + ellapsedDown).Seconds()

	c.bias += int(0.5 * float64(c.amplitude) * relativeDifference)
	c.bias = clamp(c.bias, -245, 245)
	if c.bias >= 0 {
		c.amplitude = 511 - c.bias
	} else {
		c.amplitude = 511 + c.bias
	}
	return relativeDifference
}

func (c *zeusCalibrator) calibrateHumidity() error {
	defer c.intf.Send(MakeZeusControlPoint(c.ID, 1024, 1024))
	last := time.Now()
	targetTemperature := (c.temperatureTargets.High + c.temperatureTargets.Low) / 2.0
	lastError := float32(math.NaN())
	integralError := float32(0.0)

	c.temperatureCommand, c.humidityCommand = 0, 0

	c.onNewReport = func(tr timedReport) {
		if last.Before(tr.T) == false {
			return
		}
		error := targetTemperature - tr.R.Temperature[0]
		ellapsed := float32(tr.T.Sub(last).Seconds())
		if float64(lastError) != float64(lastError) {
			lastError = error
			return
		}
		last = tr.T
		dError := (error - lastError) / ellapsed
		lastError = error
		integralError += error * ellapsed
		target := c.temperatureParameters.Kp*error + c.temperatureParameters.Kd*dError + c.temperatureParameters.Ki*integralError
		c.temperatureCommand = int16(clamp(target, -511, 511))

		c.sendCommands()
	}

	slog.Info("ramping up humidity", "RH", c.humidityTargets.High, "T", targetTemperature)
	c.bias = 0
	c.amplitude = 511
	c.humidityCommand = c.bangBangTarget(true)
	var ellapsedUp, ellapsedDown time.Duration

	Tu := func() float32 {
		return float32((ellapsedUp + ellapsedDown).Seconds())
	}

	timeLow := time.Now()
	if err := c.sendCommands(); err != nil {
		return err
	}

	timeHigh, _, err := c.waitZeusReport(
		HUMIDITY_RAMP_UP,
		func(r *arke.ZeusReport) bool {
			return r.Humidity >= c.humidityTargets.High
		})
	if err != nil {
		return err
	}
	ellapsedUp = timeHigh.Sub(timeLow)
	slog.Info("humidity reached", "RH", c.humidityTargets.High, "ellapsed", ellapsedUp)

	onTarget := 0
	for i := 0; i < c.cycles; i += 1 {
		c.resetMeasuredRange()
		log := slog.With("cycle", i)
		log.Info("drying-down", "RH", c.humidityTargets.Low, "bias", c.bias, "amplitude", c.amplitude, "T", targetTemperature)
		c.humidityCommand = c.bangBangTarget(false)
		if err := c.sendCommands(); err != nil {
			return err
		}
		var r *arke.ZeusReport
		timeLow, r, err = c.waitZeusReport(
			HUMIDITY_CHANGE,
			func(r *arke.ZeusReport) bool {
				return r.Humidity <= c.humidityTargets.Low
			})
		if err != nil {
			return err
		}
		ellapsedDown = timeLow.Sub(timeHigh)

		log.Info("humidity reached", "RH", r.Humidity, "ellapsed", ellapsedDown)
		log.Info("humidifiying up", "RH", c.humidityTargets.High, "bias", c.bias, "amplitude", c.amplitude, "T", targetTemperature)
		c.humidityCommand = c.bangBangTarget(true)
		if err := c.sendCommands(); err != nil {
			return err
		}

		timeHigh, r, err = c.waitZeusReport(
			HUMIDITY_CHANGE,
			func(r *arke.ZeusReport) bool {
				return r.Humidity >= c.humidityTargets.High
			})
		if err != nil {
			return err
		}
		ellapsedUp = timeHigh.Sub(timeLow)

		log.Info("humidity reached", "RH", r.Humidity, "ellapsed", ellapsedUp)

		relativeDifference := c.updateBangBang(ellapsedUp, ellapsedDown)
		ku := c.Ku(c.humidityMeasuredRange.Low, c.humidityMeasuredRange.High)
		log.Info("cycle result",
			"duration", ellapsedDown+ellapsedUp,
			"difference", relativeDifference,
			"Ku", ku,
			"Tu", Tu())

		if math.Abs(relativeDifference) < 0.05 {
			onTarget += 1
			if onTarget >= 2 {
				log.Info("twice on target: DONE")
				break
			}
		} else {
			onTarget = 0
		}

	}

	ku := c.Ku(c.humidityMeasuredRange.Low, c.humidityMeasuredRange.High)
	c.humidityParameters = pidParametersFromKuTu(ku, Tu())

	slog.Info("got following Humidity PID parameters",
		"Ku", ku,
		"Tu", Tu(),
		"PID", c.humidityParameters,
	)

	return nil
}
