import { Injectable } from '@angular/core';
import { Adapter } from './adapter';

import { State,StateAdapter } from './state.model';
import { Alarm,AlarmLevel,CompareAlarm,AlarmAdapter} from './alarm.model';

export class Bounds {
	constructor(public Min: number,
				public Max: number) {}
}


@Injectable({
    providedIn: 'root'
})
export class BoundsAdapter implements Adapter<Bounds> {
	adapt(item: any): Bounds {
		let res = new Bounds(0,100);
		if (item.Min != null) {
			res.Min = item.Min;
		}
		if (item.Max != null) {
			res.Max = item.Max;
		}
		return res;
	}
}

export class Zone {
	constructor(public Host: string,
				public Name: string,
				public Temperature: number,
				public TemperatureBounds: Bounds,
				public Humidity: number,
				public HumidityBounds: Bounds,
				public Alarms: Alarm[],
				public Current: State,
				public CurrentEnd: State,
				public Next: State,
				public NextEnd: State,
				public NextTime: Date) {}

    temperatureStatus() {
		if (this.Temperature < this.TemperatureBounds.Min ) {
			return 'warning';
		}
		if (this.Temperature > this.TemperatureBounds.Max ) {
			return 'danger';
		}
		return 'success';
    }

    humidityStatus() {
		if (this.Humidity < this.HumidityBounds.Min ) {
			return 'danger';
		}
		if (this.Humidity > this.HumidityBounds.Max ) {
			return 'warning';
		}
		return 'success';
    }

    alarmStatus() {
		if (this.Alarms.length == 0 ) {
			return 'danger';
		}
		let aRes = this.Alarms[0];
		for ( let a of this.Alarms ) {
			if (a.On == false ) {
				continue;
			}
			if ( aRes.On == false || a.Level > aRes.Level ) {
				aRes = a;
			}

		}
		return aRes.action();
    }

	sortedAlarm() {
		let res = Object.assign([],this.Alarms);
		return res.sort(CompareAlarm);
	}

	numberOfActiveAlarms() {
		let res = 0;
		for ( let a of this.Alarms ) {
			if (a.On == true ) {
				res += 1;
			}
		}
		return res;
	}

}


@Injectable({
    providedIn: 'root'
})
export class ZoneAdapter implements Adapter<Zone> {

	constructor(private alarmAdapter: AlarmAdapter,
				private boundsAdapter: BoundsAdapter,
				private stateAdapter: StateAdapter) {}

	adapt(item: any): Zone {
		let alarms: Alarm[] = [];
		if (item.Alarms != null) {
			for ( let a of item.Alarms ) {
				alarms.push(this.alarmAdapter.adapt(a));
			}
		}
		let current: State = null;
		let currentEnd: State = null;
		let next: State = null;
		let nextEnd: State = null;
		let nextTime: Date = null;
		if (item.Current != null) {
			current = this.stateAdapter.adapt(item.Current);
		}
		if (item.CurrentEnd != null ) {
			currentEnd = this.stateAdapter.adapt(item.CurrentEnd);
		}

		if ( item.Next != null && item.NextTime != null ) {
			next = this.stateAdapter.adapt(item.Next);
			nextTime = new Date(item.NextTime);
		}

		if ( item.NextEnd != null ) {
			nextEnd = this.stateAdapter.adapt(item.NextEnd);
		}

		return new Zone(
			item.Host,
			item.Name,
			item.Temperature,
			this.boundsAdapter.adapt(item.TemperatureBounds),
			item.Humidity,
			this.boundsAdapter.adapt(item.HumidityBounds),
			alarms,
			current,
			currentEnd,
			next,
			nextEnd,
			nextTime
		);
	}
}
