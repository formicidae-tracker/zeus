import { Component, OnInit } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ZoneService } from '../zone.service';
import { Zone }  from '../zone';

@Component({
    selector: 'app-home',
    templateUrl: './home.component.html',
    styleUrls: ['./home.component.css']
})
export class HomeComponent implements OnInit {
    zones: Zone[];

    constructor(zs : ZoneService, title: Title) {
        this.zones = zs.list();
        title.setTitle('Olympus: Home')
    }

    ngOnInit() {
    }

}
