package main

import "math"

type State struct {
	Temperature  Temperature
	Humidity     Humidity
	Wind         Wind
	VisibleLight Light
	UVLight      Light
}

func (s *State) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type StateYAML struct {
		Temperature  float64 `yaml:"temperature,omitempty"`
		Humidity     float64 `yaml:"humidity"`
		Wind         float64 `yaml:"wind"`
		VisibleLight float64 `yaml:"visible-light"`
		UVLight      float64 `yaml:"UV-light"`
	}

	res := StateYAML{}
	res.Temperature = math.Inf(-1)
	res.Humidity = math.Inf(-1)
	res.Wind = math.Inf(-1)
	res.VisibleLight = math.Inf(-1)
	res.UVLight = math.Inf(-1)

	if err := unmarshal(&res); err != nil {
		return err
	}
	s.Temperature = Temperature(res.Temperature)
	s.Humidity = Humidity(res.Humidity)
	s.Wind = Wind(res.Wind)
	s.VisibleLight = Light(res.VisibleLight)
	s.UVLight = Light(res.UVLight)
	return nil
}
