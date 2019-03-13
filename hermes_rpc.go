package dieu

import (
	"errors"
	"path"
)

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

type HermesError string

func (e HermesError) ToError() error {
	if len(e) == 0 {
		return nil
	}
	return errors.New(string(e))
}

func ReturnError(ret *HermesError, val error) {
	if val == nil {
		*ret = HermesError("")
	}
	*ret = HermesError(val.Error())
}
