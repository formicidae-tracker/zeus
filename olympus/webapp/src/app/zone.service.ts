import { Injectable } from '@angular/core';


export class Zone {
    host: string
    name: string
    temperature: number
    humidity: number
    constructor(host: string, name: string) {
        this.host = host;
        this.name = name;
        this.temperature = 21.0;
        this.humidity = 45;
    }

    temperatureStatus() {
        return 'success'
    }

    humidityStatus() {
        return 'warning'
    }

    alarmStatus() {
        return 'danger'
    }

}

@Injectable({
  providedIn: 'root'
})
export class ZoneService {
    constructor() {
    }

    list() {
        return [new Zone("helms-deep","box"),new Zone("helms-deep","tunnel"), new Zone("minas-tirith","box")];
    }
}
