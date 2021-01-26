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
