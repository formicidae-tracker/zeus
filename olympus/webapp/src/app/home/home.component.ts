import { Component, OnInit } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ZoneService } from '../zone.service';
import { Zone }  from '../core/zone.model';
import { interval } from 'rxjs';
@Component({
    selector: 'app-home',
    templateUrl: './home.component.html',
    styleUrls: ['./home.component.css']
})
export class HomeComponent implements OnInit {
    zones: Zone[];

    constructor(private zs : ZoneService, private title: Title) {
		this.zones = [];
    }

    ngOnInit() {
		this.zs.list().subscribe( (list) => {
			for( let zd of list)  {
				this.zs.getZone(zd.Host,zd.Name).subscribe( (zone) => {
					console.log(zone);
					this.zones.push(zone);
				})
			}
		});

		this.title.setTitle('Olympus: Home')

		interval(20000).subscribe(x => {
			this.zs.list().subscribe( (list) => {
				this.zones = [];
				for( let zd of list)  {
					this.zs.getZone(zd.Host,zd.Name).subscribe( (zone) => {
						console.log(zone);
						this.zones.push(zone);
					})
				}
			});
		})
    }

}
