package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"sync"
	"time"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Interface       string        `long:"interface" short:"i" description:"Interface to use" default:"slcan0"`
	ID              uint8         `long:"id" description:"ID to calibrate" default:"1"`
	Temperature     float64       `long:"temperature" short:"t" description:"calibration temperature" default:"26.0"`
	Humidity        float64       `long:"humidity" short:"u" description:"calibration humidity" default:"50.0"`
	Wind            float64       `long:"wind" short:"n" description:"calibration wind" default:"100.0"`
	Duration        time.Duration `long:"duration" short:"d" description:"time to wait to reach desired temperature" default:"2h"`
	ReferenceSensor uint8         `long:"reference-sensor" short:"r" description:"Select a sensor as reference, if 0 mean of tmp1075 is used" default:"0"`
	DryRun          bool          `long:"dry-run" short:"y" description:"dry run do no set the value at the end"`
	Window          int           `long:"window" short:"w" description:"Size of the averaging window" default:"100"`
	ResetDelta      bool          `long:"reset-delta"  description:"Resets the delta before doing anything"`
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
	if math.IsNaN(float64(value)) {
		return
	}

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

	heartbeats := make(chan *arke.HeartBeatData)
	delta := make(chan *arke.ZeusDeltaTemperature)
	averagers := []*TemperatureWindowAverager{
		NewTemperatureWindowAverager(opts.Window),
		NewTemperatureWindowAverager(opts.Window),
		NewTemperatureWindowAverager(opts.Window),
		NewTemperatureWindowAverager(opts.Window),
	}
	go func() {
		defer func() {
			close(heartbeats)
			close(delta)
		}()
		i := 0
		once := false
		messageWhiteList := map[arke.MessageClass]func(m arke.ReceivableMessage){
			arke.ZeusDeltaTemperatureMessage: func(m arke.ReceivableMessage) {
				delta <- m.(*arke.ZeusDeltaTemperature)
			},
			arke.HeartBeatMessage: func(m arke.ReceivableMessage) {
				heartbeats <- m.(*arke.HeartBeatData)
			},
			arke.ZeusReportMessage: func(m arke.ReceivableMessage) {
				c := m.(*arke.ZeusReport)
				for idx, a := range averagers {
					a.Push(c.Temperature[idx])
				}
				if i%5 == 0 {
					if once == true {
						fmt.Fprintf(os.Stderr, "\033[F\033[K")
					} else {
						once = true
					}
					fmt.Fprintf(os.Stdout, "%s : %+v\n", time.Now().Format("Mon Jan 2 15:04:05"), c)
				}
				i++
			},
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
			if treat, ok := messageWhiteList[m.MessageClassID()]; ok == true {
				treat(m)
			}
		}
	}()
	if err := intf.Send(arke.MakePing(arke.ZeusClass)); err != nil {
		return err
	}

	tick := time.NewTicker(10 * time.Second)
	select {
	case h := <-heartbeats:
		log.Printf("Found Zeus Node %d version %d.%d", opts.ID, h.MajorVersion, h.MinorVersion)
	case <-tick.C:
		tick.Stop()
		return fmt.Errorf("Ping of Zeus %d timouted", opts.ID)
	}
	tick.Stop()

	actualDelta := &arke.ZeusDeltaTemperature{
		Delta: [4]float32{0, 0, 0, 0},
	}
	if opts.ResetDelta {
		if err := arke.SendMessage(intf, actualDelta, false, arke.NodeID(opts.ID)); err != nil {
			return err
		}
	}

	if err := arke.RequestMessage(intf, actualDelta, arke.NodeID(opts.ID)); err != nil {
		return err
	}
	tick = time.NewTicker(10 * time.Second)

	select {
	case actualDelta = <-delta:
		log.Printf("Current delta are: %+v", actualDelta)
	case <-tick.C:
		tick.Stop()
		return fmt.Errorf("Fetching of zeus delta timouted")
	}
	tick.Stop()

	sp := arke.ZeusSetPoint{
		Temperature: float32(opts.Temperature),
		Humidity:    float32(opts.Humidity),
		Wind:        uint8(math.Min(math.Max(opts.Wind/100.0, 0), 1.0) * 255),
	}

	if err := arke.SendMessage(intf, &sp, false, arke.NodeID(opts.ID)); err != nil {
		return err
	}
	log.Printf("Sent %+v", sp)
	defer func() {
		intf.Send(arke.MakeResetRequest(arke.ZeusClass, arke.NodeID(opts.ID)))
		intf.Send(arke.MakeResetRequest(arke.CelaenoClass, arke.NodeID(opts.ID)))
	}()

	time.Sleep(opts.Duration)

	deltas := []float32{
		0.0, 0.0, 0.0, 0.0,
	}

	ref := float32(0.0)
	if opts.ReferenceSensor > 0 && opts.ReferenceSensor < 5 {
		ref = averagers[opts.ReferenceSensor-1].Average()
	} else {
		for i := 1; i < 4; i++ {
			ref += averagers[i].Average() / 3.0
		}
	}

	for i, a := range averagers {
		deltas[i] = ref - a.Average() + actualDelta.Delta[i]
		log.Printf("Sensor %d: Mean %.3f actual delta: %.3f new delta: %.3f ", i, a.Average(), actualDelta.Delta[i], deltas[i])
		actualDelta.Delta[i] = deltas[i]
	}

	if opts.DryRun == true {
		log.Printf("Not sending any data")
		return nil
	}

	return arke.SendMessage(intf, actualDelta, false, arke.NodeID(opts.ID))
}

func main() {

	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
