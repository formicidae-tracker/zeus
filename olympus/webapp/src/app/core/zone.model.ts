import { Injectable } from '@angular/core';
import { Adapter } from './adapter';

import {Alarm,AlarmLevel,CompareAlarm,AlarmAdapter} from './alarm.model';


export class Zone {
	constructor(public Host: string,
				public Name: string,
				public Temperature: number,
				public Humidity: number,
				public Alarms: Alarm[]) {}

    temperatureStatus() {
        return 'success';
    }

    humidityStatus() {
        return 'warning';
    }

    alarmStatus() {
		if (this.Alarms.length == 0 ) {
			return 'danger';
		}
		let aRes = this.Alarms[0];
		for ( let a of this.Alarms ) {
			if ( a.Level > aRes.Level ) {
				aRes = a;
			}

		}
		return aRes.action();
    }

	sortedAlarm() {
		let res = Object.assign([],this.Alarms);
		return res.sort(CompareAlarm);
	}

}


@Injectable({
    providedIn: 'root'
})
export class ZoneAdapter implements Adapter<Zone> {

	constructor(private alarmAdapter: AlarmAdapter) {}

	adapt(item: any): Zone {
		let alarms: Alarm[] = [];
		for ( let a of item.Alarms ) {
			alarms.push(this.alarmAdapter.adapt(a));
		}

		return new Zone(
			item.Host,
			item.Name,
			item.Temperature,
			item.Humidity,
			alarms
		);
	}
}
