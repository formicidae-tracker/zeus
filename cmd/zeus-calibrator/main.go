package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
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
		Messages: make(chan timedMessage, 10),
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
	intf.logger.Debug("sending frame", c)
	return intf.intf.Send(c)
}

func pingZeus(intf *canInterface, ID arke.NodeID) error {
	if err := intf.Send(arke.MakePing(arke.ZeusClass)); err != nil {
		return fmt.Errorf("could not send ping: %w", err)
	}

	deadline := time.After(500 * time.Millisecond)
	for {
		select {
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

func Execute() error {
	opts := Options{}

	if _, err := flags.Parse(&opts); err != nil {
		return err
	}

	intf, err := OpenCanInterface(opts.Interface)
	if err != nil {
		return err
	}
	defer intf.Close()

	ID := arke.NodeID(opts.ID)

	if err := pingZeus(intf, ID); err != nil {
		return fmt.Errorf("could not ping zeus %d: %w", opts.ID, err)
	}

}

func main() {

	if err := Execute(); err != nil {
		slog.Error("unhandled error", "error", err)
		os.Exit(1)
	}
}
