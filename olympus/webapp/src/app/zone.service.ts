import { Injectable } from '@angular/core';


export class Zone {
    host: string
    name: string
    constructor(host: string, name: string) {
        this.host = host;
        this.name = name;
    }
}

@Injectable({
  providedIn: 'root'
})
export class ZoneService {
    constructor() {
    }

    list() {
        return [new Zone("helms-deep","box"),new Zone("helms-deep","tunnel"), new Zone("minas-tirith","box"), new Zone('rivendel','box')];
    }
}
