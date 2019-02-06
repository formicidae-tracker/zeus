package main

import "time"

type Transition struct {
	From, To *State
	Duration time.Duration
	Start    time.Time
	After    time.Duration
}
