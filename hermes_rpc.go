package dieu

import "path"

type ZoneUnregistration struct {
	Host string
	Name string
}

type ZoneRegistration struct {
	Host           string
	Name           string
	MinTemperature *float64
	MaxTemperature *float64
	MinHumidity    *float64
	MaxHumidity    *float64
	Alarms         []Alarm
}

func (zr ZoneUnregistration) Fullname() string {
	return path.Join(zr.Host, "zone", zr.Name)
}

func (zr ZoneRegistration) Fullname() string {
	return path.Join(zr.Host, "zone", zr.Name)
}
