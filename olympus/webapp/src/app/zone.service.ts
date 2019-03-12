import { Injectable } from '@angular/core';
import { Zone,ZoneAdapter } from './core/zone.model';
import { Alarm,AlarmLevel } from './core/alarm.model';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { HttpClient, HttpHeaders } from '@angular/common/http';

@Injectable({
  providedIn: 'root'
})
export class ZoneService {
    constructor(private httpClient: HttpClient, private adapter : ZoneAdapter) {
    }



    list(): Zone[] {
		let t = 21.0;
		let h = 45;
		let a = [
			new Alarm("Temperature Out of Bound",false,null,AlarmLevel.Critical,0),
			new Alarm("Temperature Unreachable",true,new Date(),AlarmLevel.Warning,2),
			new Alarm("Device Zeus.1 is missing on slcan0",false,new Date(),AlarmLevel.Critical,0),
		];
        return [new Zone("helms-deep","box",t,h,a),new Zone("helms-deep","tunnel",t,h,a), new Zone("minas-tirith","box",t,h,a)];
    }

	getZone(host: string, zone: string): Observable<Zone> {
		return this.httpClient.get<any>('api/host/'+host+'/zone/'+zone).pipe(
			map(item => this.adapter.adapt(item)));
	}

}
