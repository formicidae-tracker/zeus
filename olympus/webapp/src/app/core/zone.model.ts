import { Injectable } from '@angular/core';
import { Adapter } from './adapter';

import {Alarm,AlarmLevel,CompareAlarm,AlarmAdapter} from './alarm.model';

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
				public Alarms: Alarm[]) {}

    temperatureStatus() {
		if (this.Temperature < this.TemperatureBounds.Min ) {
			return 'warning';
		}
		if (this.Temperature > this.TemperatureBounds.Max ) {
			return 'danger';
		}
		return 'sucess';
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

	constructor(private alarmAdapter: AlarmAdapter, private boundsAdapter: BoundsAdapter) {}

	adapt(item: any): Zone {
		let alarms: Alarm[] = [];
		for ( let a of item.Alarms ) {
			alarms.push(this.alarmAdapter.adapt(a));
		}
		return new Zone(
			item.Host,
			item.Name,
			item.Temperature,
			this.boundsAdapter.adapt(item.TemperatureBounds),
			item.Humidity,
			this.boundsAdapter.adapt(item.HumidityBounds),
			alarms
		);
	}
}
