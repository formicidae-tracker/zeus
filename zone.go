package dieu

type Zone struct {
	CANInterface       string      `yaml:"can-interface"`
	DevicesID          uint        `yaml:"devices-id"`
	ClimateReportFile  string      `yaml:"climate-report-file"`
	MinimalTemperature Temperature `yaml:"minimal-temperature"`
	MaximalTemperature Temperature `yaml:"maximal-temperature"`
	MinimalHumidity    Humidity    `yaml:"minimal-humidity"`
	MaximalHumidity    Humidity    `yaml:"maximal-humidity"`
	States             []State
	Transitions        []Transition
}
