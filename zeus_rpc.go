package zeus

import (
	"errors"
	"path"
	"time"
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
}

type StateReport struct {
	Zone       string
	Current    State
	CurrentEnd *State
	Next       *State
	NextEnd    *State
	NextTime   *time.Time
}

func ZoneIdentifier(host, name string) string {
	return path.Join(host, "zone", name)
}

func (zr ZoneUnregistration) Fullname() string {
	return ZoneIdentifier(zr.Host, zr.Name)
}

func (zr ZoneRegistration) Fullname() string {
	return ZoneIdentifier(zr.Host, zr.Name)
}

type ZeusError string

func (e ZeusError) ToError() error {
	if len(e) == 0 {
		return nil
	}
	return errors.New(string(e))
}
