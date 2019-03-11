import { Component, OnInit } from '@angular/core';
import { Title} from '@angular/platform-browser';
import { ActivatedRoute } from '@angular/router';

@Component({
  selector: 'app-zone',
  templateUrl: './zone.component.html',
  styleUrls: ['./zone.component.css']
})
export class ZoneComponent implements OnInit {
    zoneName: any
    hostName: any
    constructor(private route: ActivatedRoute, private title: Title) { }

    ngOnInit() {
        this.zoneName = this.route.snapshot.paramMap.get('zoneName');
        this.hostName = this.route.snapshot.paramMap.get('hostName');
        this.title.setTitle('Olympus: '+this.hostName+'.'+this.zoneName)
    }

}
