package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"git.tuleu.science/fort/libarke/src-go/arke"
	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Interface       string        `long:"interface" short:"i" description:"Interface to use" default:"slcan0"`
	ID              uint8         `long:"id" description:"ID to calibrate" default:"1"`
	Temperature     float64       `long:"temperature" short:"t" description:"calibration temperature" default:"26.0"`
	Duration        time.Duration `long:"duration" short:"d" description:"time to wait to reach desired temperature" default:"2h"`
	ReferenceSensor uint8         `long:"reference-sensor" short:"r" description:"Select a sensor as reference, if 0 mean of tmp1075 is used" default:"0"`
	DryRun          bool          `long:"dry-run" short:"y" description:"dry run do no set the value at the end"`
}

type TemperatureWindowAverager struct {
	points []float32
	idx    int
	size   int
	mx     *sync.Mutex
}

func NewTemperatureWindowAverager(size int) *TemperatureWindowAverager {
	return &TemperatureWindowAverager{
		points: make([]float32, 0, size),
		idx:    0,
		size:   size,
		mx:     &sync.Mutex{},
	}
}

func (a *TemperatureWindowAverager) Push(value float32) {
	a.mx.Lock()
	defer a.mx.Unlock()
	if len(a.points) < a.size {
		a.points = append(a.points, value)
		return
	}
	a.points[a.idx] = value
	a.idx = (a.idx + 1) % a.size
}

func (a *TemperatureWindowAverager) Average() float32 {
	res := float32(0.0)
	a.mx.Lock()
	defer a.mx.Unlock()
	for _, v := range a.points {
		res += v / float32(len(a.points))
	}
	return res
}

func Execute() error {
	opts := Options{}

	if _, err := flags.Parse(&opts); err != nil {
		return err
	}

	intf, err := socketcan.NewRawInterface(opts.Interface)
	if err != nil {
		return err
	}

	frames := make(chan arke.ReceivableMessage)
	go func() {
		defer func() {
			close(frames)
		}()

		messageWhiteList := map[arke.MessageClass]struct{}{
			arke.ZeusReportMessage: struct{}{},
			arke.HeartBeatMessage:  struct{}{},
		}

		for {
			f, err := intf.Receive()
			if err != nil {
				log.Printf("CAN Receive error: %s", err)
			}
			m, id, err := arke.ParseMessage(&f)
			if err != nil {
				log.Printf("Arke Parsing error: %s", err)
			}
			if id != arke.NodeID(opts.ID) {
				continue
			}
			if _, ok := messageWhiteList[m.MessageClassID()]; ok == false {
				continue
			}

			frames <- m
		}

	}()
	heartbeats := make(chan *arke.HeartBeatData)
	averagers := []*TemperatureWindowAverager{
		NewTemperatureWindowAverager(100),
		NewTemperatureWindowAverager(100),
		NewTemperatureWindowAverager(100),
		NewTemperatureWindowAverager(100),
	}

	go func() {
		fmt.Fprintf(os.Stdout, "No data yet \n")
		i := 0
		for {
			tick := time.NewTicker(5 * time.Second)
			select {
			case f, ok := <-frames:
				if ok == false {
					return
				}
				switch f.MessageClassID() {
				case arke.HeartBeatMessage:
					heartbeats <- f.(*arke.HeartBeatData)
				case arke.ZeusReportMessage:
					c := f.(*arke.ZeusReport)
					for idx, a := range averagers {
						a.Push(c.Temperature[idx])
					}
					if i%5 == 0 {
						fmt.Fprintf(os.Stderr, "\033[F")
						fmt.Fprintf(os.Stdout, "%s : %+v\n", time.Now().Format("Mon Jan 2 15:04:05"), c)
					}
					i++
				}
			case <-tick.C:
				panic(fmt.Sprintf("Connection to Zeus %d timeouted", opts.ID))
			}
			tick.Stop()
		}
	}()

	if err := arke.Ping(intf, arke.ZeusClass); err != nil {
		return err
	}

	<-heartbeats

	log.Printf("Found Zeus Node %d", opts.ID)

	sp := arke.ZeusSetPoint{
		Temperature: float32(opts.Temperature),
		Humidity:    float32(5.0),
	}

	if err := arke.SendMessage(intf, &sp, false, arke.NodeID(opts.ID)); err != nil {
		return err
	}
	log.Printf("Sent %+v", sp)
	defer func() {
		arke.SendResetRequest(intf, arke.ZeusClass, arke.NodeID(opts.ID))
	}()

	time.Sleep(opts.Duration)

	for i, a := range averagers {
		log.Printf("%d: %.3f", i, a.Average())
	}

	return nil
}

func main() {

	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
