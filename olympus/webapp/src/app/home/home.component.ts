import { Component, OnInit } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ZoneService } from '../zone.service';
import { Zone }  from '../core/zone.model';

@Component({
    selector: 'app-home',
    templateUrl: './home.component.html',
    styleUrls: ['./home.component.css']
})
export class HomeComponent implements OnInit {
    zones: Zone[];

    constructor(private zs : ZoneService, private title: Title) {
    }

    ngOnInit() {
		this.zones = this.zs.list();
		this.title.setTitle('Olympus: Home')
    }

}
