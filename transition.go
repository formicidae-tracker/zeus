package main

import (
	"fmt"
	"time"
)

type Transition struct {
	From, To string
	Duration time.Duration
	Start    time.Time
}

func (t *Transition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type TransitionShadow struct {
		From, To, Start string
		Duration        time.Duration
	}
	shadow := TransitionShadow{}
	if err := unmarshal(&shadow); err != nil {
		return err
	}

	if (len(shadow.From) == 0) || len(shadow.To) == 0 {
		return fmt.Errorf("'from' and 'to' fields are required")
	}

	t.From = shadow.From
	t.To = shadow.To
	t.Duration = shadow.Duration
	var err error
	t.Start, err = time.Parse("15:04", shadow.Start)
	return err
}
