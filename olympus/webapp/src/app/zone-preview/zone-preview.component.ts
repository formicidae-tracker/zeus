import { Component, OnInit, Input } from '@angular/core';
import { Zone } from '../zone.service';


@Component({
  selector: 'app-zone-preview',
  templateUrl: './zone-preview.component.html',
  styleUrls: ['./zone-preview.component.css']
})
export class ZonePreviewComponent implements OnInit {

    @Input() zone: Zone;

    constructor() { }

    ngOnInit() {
    }
}
