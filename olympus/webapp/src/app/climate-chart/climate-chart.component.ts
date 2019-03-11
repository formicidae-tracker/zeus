import { Component, OnInit, ViewChild, ElementRef } from '@angular/core';
import { Chart } from 'chart.js'


@Component({
  selector: 'app-climate-chart',
  templateUrl: './climate-chart.component.html',
  styleUrls: ['./climate-chart.component.css']
})
export class ClimateChartComponent implements OnInit {

    @ViewChild('climateChart') climateChartRef: ElementRef;
    chart: Chart;

    constructor() { }

    ngOnInit() {
        //todo display a chart
    }

}
