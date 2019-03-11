import { Injectable } from '@angular/core';
import { Zone } from './zone';


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
