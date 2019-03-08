import { Component, OnInit } from '@angular/core';
import { Zone,ZoneService } from '../zone.service';


@Component({
    selector: 'app-home',
    templateUrl: './home.component.html',
    styleUrls: ['./home.component.css']
})
export class HomeComponent implements OnInit {
    zones: Zone[];

    constructor(zs : ZoneService) {
        this.zones = zs.list();
    }

    ngOnInit() {
    }

}
