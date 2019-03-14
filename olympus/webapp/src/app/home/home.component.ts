import { Component, OnInit, OnDestroy } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ZoneService } from '../zone.service';
import { Zone }  from '../core/zone.model';
import { Subscription,timer } from 'rxjs';
@Component({
    selector: 'app-home',
    templateUrl: './home.component.html',
    styleUrls: ['./home.component.css']
})
export class HomeComponent implements OnInit,OnDestroy {
    zones: Zone[];
	update: Subscription;


    constructor(private zs : ZoneService, private title: Title) {
		this.zones = [];
    }

    ngOnInit() {
		this.title.setTitle('Olympus: Home')

		this.update = timer(0,20000).subscribe(x => {
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

	ngOnDestroy() {
		this.update.unsubscribe()
	}

}
