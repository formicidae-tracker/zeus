import { Component, OnInit, Input } from '@angular/core';
import { State }  from '../core/state.model';


@Component({
    selector: 'app-state',
    templateUrl: './state.component.html',
    styleUrls: ['./state.component.css']
})
export class StateComponent implements OnInit {
    @Input() stateA: State;
	@Input() stateB: State;
	@Input() currentTemperature: number;
	@Input() currentHumidity: number;
	@Input() displayCurrent: boolean;


	displayValue(v :number) :string {
		if (v <= -1000.0) {
			return 'n.a.';
		}
		return (Math.round(100*v)/100).toString();
	}

    constructor() {
		this.displayCurrent = false;
	}

    ngOnInit() {
    }

}
