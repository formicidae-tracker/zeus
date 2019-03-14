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

    list(): Observable<any[]> {
		return this.httpClient.get<any>('api/zones')
	}

	getZone(host: string, zone: string): Observable<Zone> {
		return this.httpClient.get<any>('api/host/'+host+'/zone/'+zone).pipe(
			map(item => {
				return this.adapter.adapt(item)
			}));
	}

}
