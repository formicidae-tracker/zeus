import { Component, OnInit, Input } from '@angular/core';

import { Alarm } from '../core/alarm.model';

@Component({
  selector: 'app-alarm',
  templateUrl: './alarm.component.html',
  styleUrls: ['./alarm.component.css']
})
export class AlarmComponent implements OnInit {

	@Input() alarm: Alarm;

	constructor() { }


	ngOnInit() {
	}

}
