package zeus

type ZoneClimate struct {
	MinimalTemperature Temperature `yaml:"minimal-temperature,omitempty"`
	MaximalTemperature Temperature `yaml:"maximal-temperature,omitempty"`
	MinimalHumidity    Humidity    `yaml:"minimal-humidity,omitempty"`
	MaximalHumidity    Humidity    `yaml:"maximal-humidity,omitempty"`
	States             []State
	Transitions        []Transition
}
