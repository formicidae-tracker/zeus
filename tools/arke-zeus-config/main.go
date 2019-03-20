package main

import (
	"fmt"
	"log"
	"time"

	"git.tuleu.science/fort/libarke/src-go/arke"
	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Interface string `long:"interface" short:"f" default:"slcan0" description:"socketcan interface to use"`
	ID        uint   `long:"id" short:"d" default:"1" description:"Node ID to use"`

	TemperatureProportionalGain     int `long:"temp-proportional-mult" default:"-1" description:"Temperature P Gain"`
	TemperatureDerivativeGain       int `long:"temp-derivative-mult" default:"-1" description:"Temperature D Gain"`
	TemperatureIntegralGain         int `long:"temp-integral-mult" default:"-1" description:"Temperature I Gain"`
	TemperatureDividerPower         int `long:"temp-divider" default:"-1" description:"Temperature PD Divider"`
	TemperatureIntegralDividerPower int `long:"temp-integral-divider" default:"-1" description:"Temperature I Divider"`

	HumidityProportionalGain     int `long:"hum-proportional-mult" default:"-1" description:"Temperature P Gain"`
	HumidityDerivativeGain       int `long:"hum-derivative-mult" default:"-1" description:"Temperature D Gain"`
	HumidityIntegralGain         int `long:"hum-integral-mult" default:"-1" description:"Temperature I Gain"`
	HumidityDividerPower         int `long:"hum-divider" default:"-1" description:"Temperature PD Divider"`
	HumidityIntegralDividerPower int `long:"hum-integral-divider" default:"-1" description:"Temperature I Divider"`
}

type modifier func(o Options, c *arke.ZeusConfig) (bool, error)

var modifiers []modifier

func isSet(v int) bool {
	return v >= 0
}

func checkGain(v int) error {
	if v < 0 || v > 255 {
		return fmt.Errorf("invalid gain %d ∉ [0;255]", v)
	}
	return nil
}

func checkDivider(v int) error {
	if v < 0 || v > 15 {
		return fmt.Errorf("invalid divider %d ∉ [0;15]", v)
	}
	return nil
}

func modifyGain(value int, target *uint8, name string) (bool, error) {
	if isSet(value) == false {
		return false, nil
	}
	if err := checkGain(value); err != nil {
		return false, fmt.Errorf("Could not set %s: %s", name, err)
	}
	*target = uint8(value)
	return true, nil
}

func init() {
	modifiers = []modifier{
		func(o Options, c *arke.ZeusConfig) (bool, error) {
			return modifyGain(o.TemperatureProportionalGain, &(c.Temperature.ProportionnalMultiplier), "temperature P gain")
		},
		func(o Options, c *arke.ZeusConfig) (bool, error) {
			return modifyGain(o.TemperatureDerivativeGain, &(c.Temperature.DerivativeMultiplier), "temperature D gain")
		},
		func(o Options, c *arke.ZeusConfig) (bool, error) {
			return modifyGain(o.TemperatureIntegralGain, &(c.Temperature.IntegralMultiplier), "temperature I gain")
		},
		func(o Options, c *arke.ZeusConfig) (bool, error) {
			return modifyGain(o.TemperatureDividerPower, &(c.Temperature.DividerPower), "temperature P/D divider")
		},
		func(o Options, c *arke.ZeusConfig) (bool, error) {
			return modifyGain(o.TemperatureIntegralDividerPower, &(c.Temperature.DividerPowerIntegral), "temperature I divider")
		},
		func(o Options, c *arke.ZeusConfig) (bool, error) {
			return modifyGain(o.HumidityProportionalGain, &(c.Humidity.ProportionnalMultiplier), "humidity P gain")
		},
		func(o Options, c *arke.ZeusConfig) (bool, error) {
			return modifyGain(o.HumidityDerivativeGain, &(c.Humidity.DerivativeMultiplier), "humidity D gain")
		},
		func(o Options, c *arke.ZeusConfig) (bool, error) {
			return modifyGain(o.HumidityIntegralGain, &(c.Humidity.IntegralMultiplier), "humidity I gain")
		},
		func(o Options, c *arke.ZeusConfig) (bool, error) {
			return modifyGain(o.HumidityDividerPower, &(c.Humidity.DividerPower), "humidity P/D divider")
		},
		func(o Options, c *arke.ZeusConfig) (bool, error) {
			return modifyGain(o.HumidityIntegralDividerPower, &(c.Humidity.DividerPowerIntegral), "humidity I divider")
		},
	}

}

func Execute() error {
	opts := Options{}

	if _, err := flags.Parse(&opts); err != nil {
		return err
	}

	if opts.ID < 1 || opts.ID > 7 {
		return fmt.Errorf("Invalid node ID %d ∉ [1,7]", opts.ID)
	}
	ID := arke.NodeID(opts.ID)

	intf, err := socketcan.NewRawInterface(opts.Interface)
	if err != nil {
		return err
	}
	defer intf.Close()

	configs := make(chan *arke.ZeusConfig, 1)
	quit := make(chan struct{})
	go func() {
		defer func() {
			close(configs)
			close(quit)
		}()
		for {
			f, err := intf.Receive()
			if err != nil {
				if socketcan.IsClosedInterfaceError(err) == true {
					log.Printf("Closed interface")
					return
				}
				log.Printf("Could not receive CAN frame: %s", err)
				continue
			}
			m, ID, err := arke.ParseMessage(&f)
			if err != nil {
				log.Printf("Could not Parse CAN frame: %s", err)
				continue
			}
			if ID != arke.NodeID(opts.ID) || m.MessageClassID() != arke.ZeusConfigMessage {
				continue
			}
			casted, ok := m.(*arke.ZeusConfig)
			if ok == false {
				log.Printf("Internal error")
			}
			configs <- casted
		}
	}()

	arke.RequestMessage(intf, &arke.ZeusConfig{}, ID)

	var current *arke.ZeusConfig
	var ok bool
	select {
	case <-time.After(10 * time.Second):
		return fmt.Errorf("Could not found %s.Zeus.%d device", opts.Interface, ID)
	case current, ok = <-configs:
		if ok == false || current == nil {
			return fmt.Errorf("Could not get ZeusConfig")
		}
	}

	log.Printf("Current configuration is %s", current)

	modified := false

	for _, modify := range modifiers {
		eff, err := modify(opts, current)
		if err != nil {
			return err
		}
		if eff == true {
			modified = true
		}
	}

	if modified == false {
		return nil
	}

	log.Printf("sending new configuration %s", current)
	err = arke.SendMessage(intf, current, false, ID)
	if err != nil {
		return err
	}
	intf.Close()
	<-quit
	return nil
}

func main() {
	if err := Execute(); err != nil {
		log.Fatalf("Unhandled error: %s", err)
	}
}
