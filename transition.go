package main

import (
	"fmt"
	"time"
)

type Transition struct {
	From, To string
	Duration time.Duration
	Start    time.Time
	After    time.Duration
}

func (t *Transition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type TransitionShadow struct {
		From, To, Start string
		Duration, After time.Duration
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
	if shadow.After != 0 && len(shadow.Start) != 0 {
		return fmt.Errorf("'start' and 'after' field are exclusive")
	}
	if shadow.After == 0 && len(shadow.Start) == 0 {
		return fmt.Errorf("either 'after' or 'start' fields are required")
	}
	if shadow.After != 0 {
		t.After = shadow.After
		t.Start = time.Unix(0, 0)
		return nil
	}
	t.After = time.Duration(0)
	var err error
	t.Start, err = time.Parse("15:04", shadow.Start)
	return err
}
