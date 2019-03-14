import { Component, OnInit, Input } from '@angular/core';
import { State }  from '../core/state.model';


@Component({
    selector: 'app-state',
    templateUrl: './state.component.html',
    styleUrls: ['./state.component.css']
})
export class StateComponent implements OnInit {
    @Input() state: State;


    constructor() { }

    ngOnInit() {
    }

}
