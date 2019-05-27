package zeus

import (
	"fmt"
	"strconv"
	"time"
)

type Transition struct {
	From, To       string
	Duration       time.Duration
	Start          time.Time
	StartTimeDelta time.Duration
	Day            int
}

func (t *Transition) Check() error {
	if len(t.From) == 0 || len(t.To) == 0 {
		return fmt.Errorf("'From' and 'To' fields are required")
	}
	if (t.Day > 0) && t.StartTimeDelta != 0 {
		return fmt.Errorf("StartTimeDelta is only available for recurring transitions (Day!=0)")
	}

	return nil
}

func (t *Transition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type TransitionShadow struct {
		From, To, Start, Day string
		Duration             time.Duration
		StartTimeDelta       time.Duration `yaml:"start-time-delta"`
	}
	shadow := TransitionShadow{}
	if err := unmarshal(&shadow); err != nil {
		return err
	}
	t.From = shadow.From
	t.To = shadow.To
	t.Duration = shadow.Duration
	t.StartTimeDelta = shadow.StartTimeDelta
	var err error
	t.Start, err = time.Parse("15:04", shadow.Start)
	if err != nil {
		return err
	}
	if len(shadow.Day) == 0 {
		t.Day = 0
	} else {
		t.Day, err = strconv.Atoi(shadow.Day)
		if err != nil {
			return err
		}
	}

	return t.Check()
}

func (t Transition) String() string {
	if t.Day == 0 {
		return fmt.Sprintf("RecurringTransition{From: %s, To: %s, Start: %s, Duration: %s}", t.From, t.To, t.Start.Format("15:04"), t.Duration)
	}
	return fmt.Sprintf("Transition{From: %s, To: %s, Start: %s, OnDay: %d, Duration: %s}", t.From, t.To, t.Start.Format("15:04"), t.Day, t.Duration)
}
