package zeus

import "github.com/formicidae-tracker/zeus/zeuspb"

type State struct {
	Name         string
	Temperature  Temperature
	Humidity     Humidity
	Wind         Wind
	VisibleLight Light
	UVLight      Light
}

func (s *State) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type stateYAML struct {
		Name         string
		Temperature  float64 `yaml:"temperature"`
		Humidity     float64 `yaml:"humidity"`
		Wind         float64 `yaml:"wind"`
		VisibleLight float64 `yaml:"visible-light"`
		UVLight      float64 `yaml:"uv-light"`
	}

	res := stateYAML{}
	res.Temperature = UndefinedTemperature.Value()
	res.Humidity = UndefinedHumidity.Value()
	res.Wind = UndefinedWind.Value()
	res.VisibleLight = UndefinedLight.Value()
	res.UVLight = UndefinedLight.Value()

	if err := unmarshal(&res); err != nil {
		return err
	}
	s.Name = res.Name
	s.Temperature = Temperature(res.Temperature)
	s.Humidity = Humidity(res.Humidity)
	s.Wind = Wind(res.Wind)
	s.VisibleLight = Light(res.VisibleLight)
	s.UVLight = Light(res.UVLight)
	return nil
}

func (s State) MarshalYAML() (interface{}, error) {
	type saveState struct {
		Name   string
		Values map[string]float64 `yaml:",inline"`
	}
	res := saveState{
		Name:   s.Name,
		Values: make(map[string]float64),
	}
	if !IsUndefined(s.Temperature) {
		res.Values["temperature"] = float64(s.Temperature)
	}
	if !IsUndefined(s.Humidity) {
		res.Values["humidity"] = float64(s.Humidity)
	}
	if !IsUndefined(s.Wind) {
		res.Values["wind"] = float64(s.Wind)
	}
	if !IsUndefined(s.VisibleLight) {
		res.Values["visible-light"] = float64(s.VisibleLight)
	}
	if !IsUndefined(s.UVLight) {
		res.Values["uv-light"] = float64(s.UVLight)
	}
	return res, nil
}

func (s State) AsPbTarget() *zeuspb.Target {
	return &zeuspb.Target{
		Name:         s.Name,
		Temperature:  AsFloat32Pointer(s.Temperature),
		Humidity:     AsFloat32Pointer(s.Humidity),
		Wind:         AsFloat32Pointer(s.Wind),
		VisibleLight: AsFloat32Pointer(s.VisibleLight),
		UvLight:      AsFloat32Pointer(s.UVLight),
	}
}
