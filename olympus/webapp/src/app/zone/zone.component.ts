import { Component, OnInit } from '@angular/core';
import { Title} from '@angular/platform-browser';
import { ActivatedRoute } from '@angular/router';
import { Zone } from '../zone';
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
	}

    ngOnInit() {
        this.zoneName = this.route.snapshot.paramMap.get('zoneName');
        this.hostName = this.route.snapshot.paramMap.get('hostName');




		this.zoneService.getZone(this.hostName,this.zoneName)
			.subscribe(function(zone: any) {
				console.log('cououc',zone)
				this.zone = new Zone('','')
				Object.assign(this.zone,zone);
			});

        this.title.setTitle('Olympus: '+this.hostName+'.'+this.zoneName)
		interval(2000).subscribe(x => {
			this.zone.alarms[0].On = !this.zone.alarms[0].On;
			this.zone.alarms[0].LastChange = new Date();
			if ( this.zone.alarms[0].On == true ) {
				this.zone.alarms[0].Triggers += 1;
			}
		});

    }

}
