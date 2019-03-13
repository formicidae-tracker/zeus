import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';



@Injectable({
  providedIn: 'root'
})
export class ClimateReportService {

	constructor(private httpClient: HttpClient) { }

	getReport(host: string, zone: string, window: string): Observable<any> {
		console.log(host,zone,window);
		return this.httpClient.get<any>('api/host/'+host+'/zone/'+zone+'/climate-report?window='+window);
	}

}
