package main

import "time"

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
	t.From = shadow.From
	t.To = shadow.To
	t.Duration = shadow.Duration
	t.After = shadow.After
	var err error
	t.Start, err = time.Parse("15:04", shadow.Start)
	return err
}
