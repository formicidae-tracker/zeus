package zeus

import "time"

type ZeusStartArgs struct {
	Season  SeasonFile
	Version string
}

type ZeusZoneStatus struct {
	State       State
	Temperature float64
	Humidity    float64
}

type ZeusStatusReply struct {
	Running bool
	Since   time.Time
	Version string
	Zones   map[string]ZeusZoneStatus
}

type ZeusLogArgs struct {
	ZoneName   string
	Start, End int
}

type ZeusClimateLogReply struct {
	Data []ClimateReport
}

type ZeusAlarmLogReply struct {
	Data []AlarmEvent
}
