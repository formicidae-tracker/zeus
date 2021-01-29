package zeus

import (
	"path"
	"time"
)

type ZoneUnregistration struct {
	Host string
	Name string
}

type ZoneRegistration struct {
	Host                         string
	Name                         string
	MinTemperature               *float64
	MaxTemperature               *float64
	MinHumidity                  *float64
	MaxHumidity                  *float64
	NumAux                       int
	RPCAddress                   string
	SizeClimateLog, SizeAlarmLog int
}

type StateReport struct {
	ZoneIdentifier string
	Current        State
	CurrentEnd     *State
	Next           *State
	NextEnd        *State
	NextTime       *time.Time
}

func ZoneIdentifier(host, name string) string {
	return path.Join(host, "zone", name)
}

func (zr ZoneUnregistration) ZoneIdentifer() string {
	return ZoneIdentifier(zr.Host, zr.Name)
}

func (zr ZoneRegistration) ZoneIdentifier() string {
	return ZoneIdentifier(zr.Host, zr.Name)
}
