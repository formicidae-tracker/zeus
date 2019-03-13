import { Component, OnInit } from '@angular/core';
import { Title} from '@angular/platform-browser';
import { ActivatedRoute } from '@angular/router';
import { Bounds,Zone } from '../core/zone.model';
import { interval } from 'rxjs';
import { ZoneService } from '../zone.service';

@Component({
  selector: 'app-zone',
  templateUrl: './zone.component.html',
  styleUrls: ['./zone.component.css']
})

export class ZoneComponent implements OnInit {
    zoneName: string
    hostName: string
	zone: Zone
    constructor(private route: ActivatedRoute, private title: Title, private zoneService: ZoneService) {
		this.zone = new Zone("","",0,new Bounds(0,100),0,new Bounds(0,100),[]);
	}

    ngOnInit() {
        this.zoneName = this.route.snapshot.paramMap.get('zoneName');
        this.hostName = this.route.snapshot.paramMap.get('hostName');

		this.zoneService.getZone(this.hostName,this.zoneName)
			.subscribe( (zone) => {
				this.zone = zone;
			});

        this.title.setTitle('Olympus: '+this.hostName+'.'+this.zoneName)
		interval(20000).subscribe(x => {
			this.zoneService.getZone(this.hostName,this.zoneName)
				.subscribe( (zone) => {
					this.zone = zone;
				});
		});

    }

}
