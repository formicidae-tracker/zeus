package zeus

import "time"

type ZeusStartArgs struct {
	Season  SeasonFile
	Version string
}

type ZeusStatusReply struct {
	Running bool
	Since   time.Time
	Version string
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
