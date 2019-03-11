import {Alarm,AlarmLevel,CompareAlarm} from './alarm';


export class Zone {
    host: string;
    name: string;
    temperature: number;
    humidity: number;
	alarms: Alarm[];

	constructor(host: string, name: string) {
        this.host = host;
        this.name = name;
        this.temperature = 21.0;
        this.humidity = 45;
		this.alarms = [
			new Alarm('Temperature Unreachable',false,AlarmLevel.Warning),
			new Alarm('Temperature Out of Bound',false,AlarmLevel.Critical),
			new Alarm("Device Zeus.1 on 'slcan0' is missing",false,AlarmLevel.Critical)
		];

		this.alarms[1].On = true
		this.alarms[1].LastChange = new Date();
		this.alarms[1].Triggers = 2;
    }

    temperatureStatus() {
        return 'success';
    }

    humidityStatus() {
        return 'warning';
    }

    alarmStatus() {
		if (this.alarms.length == 0 ) {
			return 'danger';
		}
		let aRes = this.alarms[0];
		for ( let a of this.alarms ) {
			if ( a.Level > aRes.Level ) {
				aRes = a;
			}

		}
		return aRes.action();
    }

	sortedAlarm() {
		let res = Object.assign([],this.alarms);
		return res.sort(CompareAlarm);
	}

}
