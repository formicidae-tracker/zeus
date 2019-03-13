package main

import (
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"git.tuleu.science/fort/dieu"
)

type Point struct {
	X float64
	Y float64
}

type ClimateReportTimeSerie struct {
	Humidity        []Point
	TemperatureAnt  []Point
	TemperatureAux1 []Point
	TemperatureAux2 []Point
	TemperatureAux3 []Point
}

type ClimateReportManager interface {
	Sample()
	Inbound() chan<- dieu.ClimateReport
	LastHour() ClimateReportTimeSerie
	LastDay() ClimateReportTimeSerie
	LastWeek() ClimateReportTimeSerie
}

type window int

const (
	hour window = iota
	day
	week
)

type request struct {
	w      window
	result chan<- ClimateReportTimeSerie
}

type climateReportManager struct {
	inbound         chan dieu.ClimateReport
	requests        chan request
	quit            chan struct{}
	wg              sync.WaitGroup
	hour, day, week ClimateReportTimeSerie
	start           *time.Time
}

func truncate(ts ClimateReportTimeSerie, duration time.Duration) ClimateReportTimeSerie {
	if len(ts.Humidity) == 0 {
		return ts
	}

	idx := 0
	ellapsed := ts.Humidity[len(ts.Humidity)-1].X
	for {
		start := ts.Humidity[idx].X

		if ellapsed-start > float64(duration.Nanoseconds()/1e6) {
			idx += 1
		} else {
			break
		}
	}

	if idx != 0 {
		ts.Humidity = ts.Humidity[idx:]
		ts.TemperatureAnt = ts.TemperatureAnt[idx:]
		ts.TemperatureAux1 = ts.TemperatureAux1[idx:]
		ts.TemperatureAux2 = ts.TemperatureAux2[idx:]
		ts.TemperatureAux3 = ts.TemperatureAux3[idx:]
	}
	return ts
}

func appendPoints(ts ClimateReportTimeSerie, humidity, temperatureAnt, temperatureAux1, temperatureAux2, temperatureAux3 Point) ClimateReportTimeSerie {
	ts.Humidity = append(ts.Humidity, humidity)
	ts.TemperatureAnt = append(ts.TemperatureAnt, temperatureAnt)
	ts.TemperatureAux1 = append(ts.TemperatureAux1, temperatureAux1)
	ts.TemperatureAux2 = append(ts.TemperatureAux2, temperatureAux2)
	ts.TemperatureAux3 = append(ts.TemperatureAux3, temperatureAux3)
	return ts
}

func downsample(ts ClimateReportTimeSerie, duration time.Duration) ClimateReportTimeSerie {
	res := ClimateReportTimeSerie{make([]Point, 0, len(ts.Humidity)), make([]Point, 0, len(ts.Humidity)), make([]Point, 0, len(ts.Humidity)), make([]Point, 0, len(ts.Humidity)), make([]Point, 0, len(ts.Humidity))}

	if len(ts.Humidity) == 0 {
		return res
	}
	res = appendPoints(res, ts.Humidity[0], ts.TemperatureAnt[0], ts.TemperatureAux1[0], ts.TemperatureAux2[0], ts.TemperatureAux3[0])
	lastTime := ts.Humidity[0].X
	for i := 1; i < (len(ts.Humidity) - 1); i++ {
		curTime := ts.Humidity[i].X
		ellapsed := curTime - lastTime
		if ellapsed < float64(duration.Nanoseconds()/1e6) {

			continue
		}
		res = appendPoints(res, ts.Humidity[i], ts.TemperatureAnt[i], ts.TemperatureAux1[i], ts.TemperatureAux2[i], ts.TemperatureAux3[i])
		lastTime = curTime
	}
	if len(ts.Humidity) > 1 {
		lastIdx := len(ts.Humidity) - 1
		res = appendPoints(res, ts.Humidity[lastIdx], ts.TemperatureAnt[lastIdx], ts.TemperatureAux1[lastIdx], ts.TemperatureAux2[lastIdx], ts.TemperatureAux3[lastIdx])
	}
	return res
}

func (m *climateReportManager) addReportUnsafe(r *dieu.ClimateReport) {
	if m.start == nil {
		m.start = &time.Time{}
		*m.start = r.Time
	}

	ellapsed := float64(r.Time.Sub(*m.start).Nanoseconds() / 1e6)

	m.hour = appendPoints(m.hour,
		Point{X: ellapsed, Y: float64(r.Humidity)},
		Point{X: ellapsed, Y: float64(r.Temperatures[0])},
		Point{X: ellapsed, Y: float64(r.Temperatures[1])},
		Point{X: ellapsed, Y: float64(r.Temperatures[2])},
		Point{X: ellapsed, Y: float64(r.Temperatures[3])})

	m.day = appendPoints(m.day,
		Point{X: ellapsed, Y: float64(r.Humidity)},
		Point{X: ellapsed, Y: float64(r.Temperatures[0])},
		Point{X: ellapsed, Y: float64(r.Temperatures[1])},
		Point{X: ellapsed, Y: float64(r.Temperatures[2])},
		Point{X: ellapsed, Y: float64(r.Temperatures[3])})

	m.week = appendPoints(m.week,
		Point{X: ellapsed, Y: float64(r.Humidity)},
		Point{X: ellapsed, Y: float64(r.Temperatures[0])},
		Point{X: ellapsed, Y: float64(r.Temperatures[1])},
		Point{X: ellapsed, Y: float64(r.Temperatures[2])},
		Point{X: ellapsed, Y: float64(r.Temperatures[3])})

	m.hour = downsample(m.hour, 5*time.Second)
	m.hour = truncate(m.hour, 1*time.Hour)
	m.day = downsample(m.day, 5*time.Minute)
	m.day = truncate(m.day, 24*time.Hour)

	m.week = downsample(m.week, 30*time.Minute)
	m.week = truncate(m.week, 7*24*time.Hour)
}

func (m *climateReportManager) Sample() {
	m.quit = make(chan struct{})
	defer func() {
		close(m.quit)
		m.wg.Wait()
	}()
	for {
		select {
		case r := <-m.requests:
			log.Printf("request")
			switch r.w {
			case hour:
				r.result <- m.hour
			case day:
				r.result <- m.day
			case week:
				r.result <- m.week
			default:
				r.result <- ClimateReportTimeSerie{}
			}
		case r, ok := <-m.inbound:
			if ok == false {
				return
			}
			m.addReportUnsafe(&r)
		}
	}
}

func (m *climateReportManager) Inbound() chan<- dieu.ClimateReport {
	return m.inbound
}

func (m *climateReportManager) lastReport(w window) ClimateReportTimeSerie {
	m.wg.Add(1)
	res := make(chan ClimateReportTimeSerie)
	defer func() {
		close(res)
		m.wg.Done()
	}()
	go func() {
		log.Printf("sending request")

		m.requests <- request{w: w, result: res}
	}()
	select {
	case <-m.quit:
		return ClimateReportTimeSerie{}
	case r := <-res:
		return r
	}
}

func (m *climateReportManager) LastHour() ClimateReportTimeSerie {
	return m.lastReport(hour)
}

func (m *climateReportManager) LastDay() ClimateReportTimeSerie {
	return m.lastReport(day)
}

func (m *climateReportManager) LastWeek() ClimateReportTimeSerie {
	return m.lastReport(week)
}

func NewClimateReportManager() (ClimateReportManager, error) {
	return &climateReportManager{
		inbound:  make(chan dieu.ClimateReport),
		requests: make(chan request),
	}, nil
}

var stubClimateReporter ClimateReportManager

func setClimateReporterStub() {

	stubClimateReporter, _ = NewClimateReportManager()
	end := time.Now()
	start := end.Add(-7 * 24 * time.Hour)
	go stubClimateReporter.Sample()
	go func() {
		for t := start; t.Before(end); t = t.Add(500 * time.Millisecond) {
			ellapsed := t.Sub(start).Seconds()

			toAdd := dieu.ClimateReport{
				Time:     t,
				Humidity: dieu.Humidity(40.0 + 3*math.Cos(2*math.Pi/200.0*ellapsed) + 0.5*rand.NormFloat64()),
				Temperatures: [4]dieu.Temperature{
					dieu.Temperature(20.0 + 0.5*math.Cos(2*math.Pi/1800.0*ellapsed) + 0.1*rand.NormFloat64()),
					dieu.Temperature(20.5 + 0.5*math.Cos(2*math.Pi/1800.0*ellapsed) + 0.1*rand.NormFloat64()),
					dieu.Temperature(21.0 + 0.5*math.Cos(2*math.Pi/1800.0*ellapsed) + 0.1*rand.NormFloat64()),
					dieu.Temperature(21.5 + 0.5*math.Cos(2*math.Pi/1800.0*ellapsed) + 0.1*rand.NormFloat64()),
				},
			}
			stubClimateReporter.Inbound() <- toAdd
		}
		log.Printf("done")
	}()
}
