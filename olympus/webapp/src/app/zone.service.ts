import { Injectable } from '@angular/core';
import { Zone } from './zone';
import { Observable } from 'rxjs';
import { HttpClient, HttpHeaders } from '@angular/common/http';

@Injectable({
  providedIn: 'root'
})
export class ZoneService {
    constructor(private httpClient: HttpClient) {
    }

    list() {
        return [new Zone("helms-deep","box"),new Zone("helms-deep","tunnel"), new Zone("minas-tirith","box")];
    }

	getZone(host: string, zone: string): Observable<Zone> {
		return this.httpClient.get<Zone>('api/zone/'+host+'.'+zone);
	}

}
