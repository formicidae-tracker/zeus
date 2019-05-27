package zeus

type State struct {
	Name         string
	Temperature  Temperature
	Humidity     Humidity
	Wind         Wind
	VisibleLight Light
	UVLight      Light
}

func (s *State) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type StateYAML struct {
		Name         string
		Temperature  float64 `yaml:"temperature,omitempty"`
		Humidity     float64 `yaml:"humidity"`
		Wind         float64 `yaml:"wind"`
		VisibleLight float64 `yaml:"visible-light"`
		UVLight      float64 `yaml:"uv-light"`
	}

	res := StateYAML{}
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
