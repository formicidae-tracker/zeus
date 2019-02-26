package main

type AlarmNotifier struct {
	incoming        chan Alarm
	outgoingRequest chan chan Alarm
}

func NewAlarmNotifier() *AlarmNotifier {
	return &AlarmNotifier{
		incoming:        make(chan Alarm),
		outgoingRequest: make(chan chan Alarm),
	}
}

func (d *AlarmNotifier) Dispatch() {
	outgoing := []chan Alarm{}
	defer func() {
		for _, o := range outgoing {
			close(o)
		}
	}()
	for {
		select {
		case a, ok := <-d.incoming:
			if ok == false {
				return
			}
			toDelete := []int{}
			for idx, c := range outgoing {
				select {
				case c <- a:
					continue
				default:
					toDelete = append([]int{idx}, toDelete...)
				}
			}
			for _, idx := range toDelete {
				close(outgoing[idx])
				outgoing = append(outgoing[0:idx], outgoing[(idx+1):]...)
			}
		case r, ok := <-d.outgoingRequest:
			if ok == false {
				return
			}
			outgoing = append(outgoing, r)
		}
	}
}

func (d *AlarmNotifier) Subscribe() <-chan Alarm {
	res := make(chan Alarm, 10)
	d.outgoingRequest <- res
	return res
}

func (d *AlarmNotifier) Close() {
	close(d.incoming)
	close(d.outgoingRequest)
}

func (d *AlarmNotifier) Notify() chan<- Alarm {
	return d.incoming
}
