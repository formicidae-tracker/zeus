package zeus

import (
	"fmt"
	"math"
	"sort"
	"time"
)

func SanitizeUnit(u BoundedUnit) float64 {
	if IsUndefined(u) || math.IsNaN(u.Value()) {
		return -1000.0
	}
	return u.Value()
}

func SanitizeState(s State) State {
	return State{
		Name:         s.Name,
		Temperature:  Temperature(SanitizeUnit(s.Temperature)),
		Humidity:     Humidity(SanitizeUnit(s.Humidity)),
		Wind:         Wind(SanitizeUnit(s.Wind)),
		VisibleLight: Light(SanitizeUnit(s.VisibleLight)),
		UVLight:      Light(SanitizeUnit(s.UVLight)),
	}
}

type Interpolation interface {
	State(t time.Time) State
	String() string
	End() *State
}

type staticClimate State

func (s *staticClimate) State(time.Time) State {
	return State(*s)
}

func (s *staticClimate) String() string {
	return fmt.Sprintf("static state: %+v", State(*s))
}

func (s *staticClimate) End() *State {
	return nil
}

type climateTransition struct {
	start    time.Time
	from, to State
	duration time.Duration
}

func interpolate(from, to, completion float64) float64 {
	if from == math.Inf(-1) {
		if to == math.Inf(-1) {
			return math.Inf(-1)
		}
		return to
	} else if to == math.Inf(-1) {
		return from
	}
	return from + (to-from)*completion
}

func interpolateState(from, to State, completion float64) State {
	return State{
		Name:         fmt.Sprintf("%s to %s", from.Name, to.Name),
		Temperature:  Temperature(interpolate(from.Temperature.Value(), to.Temperature.Value(), completion)),
		Humidity:     Humidity(interpolate(from.Humidity.Value(), to.Humidity.Value(), completion)),
		Wind:         Wind(interpolate(from.Wind.Value(), to.Wind.Value(), completion)),
		VisibleLight: Light(interpolate(from.VisibleLight.Value(), to.VisibleLight.Value(), completion)),
		UVLight:      Light(interpolate(from.UVLight.Value(), to.UVLight.Value(), completion)),
	}

}

func (i *climateTransition) State(t time.Time) State {
	ellapsed := t.Sub(i.start)
	if ellapsed < 0 {
		ellapsed = 0
	} else if ellapsed > i.duration {
		ellapsed = i.duration
	}
	completion := float64(ellapsed.Seconds()) / float64(i.duration.Seconds())
	return interpolateState(i.from, i.to, completion)
}

func (i *climateTransition) String() string {
	return fmt.Sprintf("transition from '%s' to '%s' in %s at %s", i.from.Name, i.to.Name, i.duration, i.start)
}

func (t *climateTransition) End() *State {
	state := SanitizeState(t.to)
	return &state
}

type ClimateInterpoler interface {
	CurrentInterpolation(t time.Time) (Interpolation, time.Time, Interpolation)
}

type computedState struct {
	State
	transitionForward  []Transition
	transitionBackward []Transition
}

type climateInterpolation struct {
	current          *computedState
	previous         *computedState
	states           map[string]*computedState
	currentTime      time.Time
	year, month, day int
}

type computedTransition struct {
	time       time.Time
	transition Transition
}

type computedTransitionList []computedTransition

func (l computedTransitionList) Len() int {
	return len(l)
}

func (l computedTransitionList) Less(i, j int) bool {
	return l[i].time.Before(l[j].time)
}

func (l computedTransitionList) Swap(i, j int) {
	l[j], l[i] = l[i], l[j]
}

func (i *climateInterpolation) computeTransitions(t time.Time, forward bool) []computedTransition {
	//gets t date
	y, m, d := t.Date()

	res := map[time.Time][]computedTransition{}
	var transitions []Transition
	if forward == true {
		transitions = i.current.transitionForward
	} else {
		transitions = i.current.transitionBackward
	}
	for _, tr := range transitions {
		if forward == false && i.previous != nil && tr.From != i.previous.Name {
			continue
		}
		if tr.Day != 0 {
			trigger := tr.Start.AddDate(i.year, i.month-1, i.day-1+tr.Day-1)
			if (forward == true && trigger.Before(t)) || (forward == false && trigger.After(t)) {
				continue
			}
			res[trigger] = append(res[trigger], computedTransition{trigger, tr})
		} else {
			tUTC := t.UTC()
			dayEllapsed := int(tUTC.Sub(time.Date(i.year, time.Month(i.month), i.day, tUTC.Hour(), tUTC.Minute(), tUTC.Second(), tUTC.Nanosecond(), time.UTC)) / (24 * time.Hour))
			trigger := tr.Start.AddDate(y, int(m)-1, d-1).Add(tr.StartTimeDelta * time.Duration(dayEllapsed))
			if forward == true && trigger.Before(t) {
				trigger = trigger.AddDate(0, 0, 1)
			}
			if forward == false && trigger.After(t) {
				trigger = trigger.AddDate(0, 0, -1)
			}
			res[trigger] = append(res[trigger], computedTransition{trigger, tr})
		}
	}

	//non - recuring tranistion have priority on reccurent one
	resList := make([]computedTransition, 0, len(res))
	for _, t := range res {
		if len(t) == 0 {
			continue
		}
		if len(t) == 1 || t[0].transition.Day != 0 {
			resList = append(resList, t[0])
			continue
		}
		resList = append(resList, t[1])
		//we simply ignore multiple recurring transition
	}

	return resList
}

func (i *climateInterpolation) nextForwardTransition(t time.Time) (computedTransition, bool) {
	orderedTransitions := i.computeTransitions(t, true)
	if len(orderedTransitions) == 0 {
		return computedTransition{}, false
	}
	sort.Sort(computedTransitionList(orderedTransitions))
	return orderedTransitions[0], true
}

func (i *climateInterpolation) previousBackwardTransition(t time.Time) (computedTransition, bool) {
	orderedTransitions := i.computeTransitions(t, false)
	if len(orderedTransitions) == 0 {
		return computedTransition{}, false
	}
	sort.Sort(sort.Reverse(computedTransitionList(orderedTransitions)))
	return orderedTransitions[0], true
}

func (i *climateInterpolation) walkTo(t time.Time) (prev, next computedTransition, prevOK, nextOK bool) {
	for {
		//		log.Printf("current state is %+v", i.current.State)
		prev, prevOK = i.previousBackwardTransition(i.currentTime)
		//		log.Printf("prev: %s to %s at %s %v %s", prev.transition.From, prev.transition.To, prev.time, prevOK, t)
		if prevOK == true && t.Before(prev.time) {
			//			log.Printf("moving to %s", prev.transition.From)
			i.previous = i.current
			i.current = i.states[prev.transition.From]
			i.currentTime = prev.time
			continue
		}

		next, nextOK = i.nextForwardTransition(i.currentTime)
		//		log.Printf("next: %s to %s at %s %v %s", next.transition.From, next.transition.To, next.time, prevOK, t)
		if nextOK == true && t.After(next.time) {
			//			log.Printf("moving to %s", next.transition.To)
			i.previous = i.current
			i.current = i.states[next.transition.To]
			i.currentTime = next.time
			continue
		}
		//		log.Printf("Its current state")
		i.currentTime = t
		return prev, next, prevOK, nextOK
	}
}

func (i *climateInterpolation) CurrentInterpolation(t time.Time) (Interpolation, time.Time, Interpolation) {
	prevT, nextT, prevOK, nextOK := i.walkTo(t.UTC())
	var currentI, nextI Interpolation
	var nextTime time.Time
	if prevOK == false || t.After(prevT.time.Add(prevT.transition.Duration)) {
		currentI = (*staticClimate)(&(i.current.State))
		if nextOK == true {
			nextI = &climateTransition{
				start:    nextT.time,
				from:     i.current.State,
				to:       i.states[nextT.transition.To].State,
				duration: nextT.transition.Duration,
			}
			nextTime = nextT.time
		}
	} else {
		currentI = &climateTransition{
			start:    prevT.time,
			from:     i.states[prevT.transition.From].State,
			to:       i.current.State,
			duration: prevT.transition.Duration,
		}
		nextI = (*staticClimate)(&(i.current.State))
		nextTime = prevT.time.Add(prevT.transition.Duration)
	}
	return currentI, nextTime, nextI
}

func NewClimateInterpoler(states []State, transitions []Transition, reference time.Time) (ClimateInterpoler, error) {
	if len(states) == 0 {
		return nil, fmt.Errorf("climate interpolation needs at least one state")
	}
	y, m, d := reference.Date()
	res := &climateInterpolation{
		states:      make(map[string]*computedState),
		year:        y,
		month:       int(m),
		day:         d,
		currentTime: reference.AddDate(0, 0, -2),
	}
	for _, s := range states {
		if _, ok := res.states[s.Name]; ok == true {
			return nil, fmt.Errorf("Cannot redefine state '%s'", s.Name)
		}
		res.states[s.Name] = &computedState{
			State:              s,
			transitionBackward: make([]Transition, 0, len(transitions)),
			transitionForward:  make([]Transition, 0, len(transitions)),
		}

		if res.current == nil {
			res.current = res.states[s.Name]
		}
	}

	for _, t := range transitions {
		to, ok := res.states[t.To]
		if ok == false {
			return nil, fmt.Errorf("Undefined state '%s' in %s", t.To, t)
		}
		from, ok := res.states[t.From]
		if ok == false {
			return nil, fmt.Errorf("Undefined state '%s' in %s", t.From, t)
		}
		to.transitionBackward = append(to.transitionBackward, t)
		from.transitionForward = append(from.transitionForward, t)
	}

	// computes the states
	for _, s := range states {
		cs := res.states[s.Name]
		if len(cs.transitionForward) != 0 {
			cs.State = interpolateState(cs.State, res.states[cs.transitionForward[0].To].State, 0)
			cs.State.Name = s.Name
		}
		if len(cs.transitionBackward) != 0 {
			cs.State = interpolateState(res.states[cs.transitionBackward[0].From].State, cs.State, 1)
			cs.State.Name = s.Name
		}
		for i, trA := range cs.transitionForward {
			for _, trB := range cs.transitionForward[i:] {
				if trA.Day != 0 && trB.Day == trA.Day {
					if trA.Start.Before(trB.Start) {
						return nil, fmt.Errorf("%s is shadowed by %s", trB, trA)
					} else if trB.Start.Before(trA.Start) {
						return nil, fmt.Errorf("%s is shadowed by %s", trA, trB)
					}
				}
			}

		}

	}

	return res, nil
}
