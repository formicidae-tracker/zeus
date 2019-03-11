export enum AlarmLevel {
	Warning = 1,
	Critical
}

export class Alarm {
	Reason: string
	On: boolean
	LastChange: Date
	Level: AlarmLevel
	Triggers: number

	constructor(r :string,on: boolean,level: AlarmLevel) {
		this.Reason = r;
		this.LastChange = null;
		this.On = on;
		this.Level = level;
		this.Triggers = 0;
	}

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
