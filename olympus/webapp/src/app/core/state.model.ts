import { Injectable } from '@angular/core';
import { Adapter } from './adapter';



export class State {
	constructor(public Name: string,
				public Humidity: number,
			    public Temperature: number,
				public Wind: number,
				public VisibleLight: number,
				public UVLight: number){}
}



@Injectable({
    providedIn: 'root'
})
export class StateAdapter implements Adapter<State> {
	adapt(item: any): State {
		return new State(
			item.Name,
			item.Humidity,
			item.Temperature,
			item.Wind,
			item.VisibleLight,
			item.UVLight);
	}
}
