import { Component, OnInit } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { Zone,ZoneService } from '../zone.service';


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
