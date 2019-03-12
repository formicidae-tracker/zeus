import { Injectable } from '@angular/core';
import { Adapter } from './adapter';

export enum AlarmLevel {
	Warning = 1,
	Critical
}

export class Alarm {

	constructor(public Reason :string,
				public On: boolean,
				public LastChange: Date,
				public Level: AlarmLevel,
				public Triggers: number
			   ) {}

	action() {
		if (this.On == false) {
			return 'info';
		}
		if (this.Level == AlarmLevel.Warning) {
			return 'warning';
		}
		return 'danger';
	}
}





export function CompareAlarm(a :Alarm, b :Alarm){
	if (a.On ==  b.On ) {
		if (a.Level < b.Level) {
			return 1;
		} else if (a.Level > b.Level) {
			return -1;
		}
		return a.Reason.localeCompare(b.Reason);
	}
	if (a.On == true) {
		return -1;
	}
	return 1;
}



@Injectable({
    providedIn: 'root'
})
export class AlarmAdapter implements Adapter<Alarm> {
	adapt(item: any): Alarm {
		return new Alarm(
			item.Reason,
			item.On,
			new Date(item.LastChange),
			item.Level,
			item.Triggers
		);
	}
}
