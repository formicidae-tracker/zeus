import { Component, AfterViewInit, OnInit, ElementRef,ViewChild, Input} from '@angular/core';
import ResizeObserver from 'resize-observer-polyfill';
import { Chart } from 'chart.js'
import { ClimateReportService } from '../climate-report.service';
import { interval } from 'rxjs';

export enum TimeWindow {
	Week = 1,
	Day,
	Hour
}

@Component({
  selector: 'app-climate-chart',
  templateUrl: './climate-chart.component.html',
  styleUrls: ['./climate-chart.component.css']
})

export class ClimateChartComponent implements AfterViewInit,OnInit {

	public Window = TimeWindow;

	timeWindow: TimeWindow;

    canvas: any;
	ctx: any;
	chart : any;

	@ViewChild('climateChartMonitor')
	public monitor: ElementRef

	@Input() hostName: string;
	@Input() zoneName: string;

	constructor(private climateReport: ClimateReportService) {
		this.timeWindow = TimeWindow.Hour;
	}

	ngOnInit() {

		this.updateChart();
		interval(10000).subscribe(x => {
			this.updateChart();
		})
	}

	ngAfterViewInit() {
		let ro = new ResizeObserver(entries => {
			for ( let e of entries) {
				const cr = e.contentRect;
				this.chart.options.width = cr.width;
				this.chart.options.height = cr.height;
				this.chart.resize();
			}
		});
		Chart.defaults.global.elements.point.radius = 0;
		Chart.defaults.global.elements.point.hitRadius = 3;

		ro.observe(this.monitor.nativeElement);
		this.canvas = document.getElementById('climateChart');
		this.ctx = this.canvas.getContext('2d');
		this.chart = new Chart(this.ctx,{
			type: 'scatter',
			data: {
				datasets: [
					{
						borderColor: '#1f77b4',
						label: 'Humidity',
						fill: false,
						showLine: true,
						lineTension: 0,
						data: [
							{x:0,y:40},
							{x:1,y:42},
							{x:2,y:38},
							{x:3,y:50},
							{x:4,y:50.2}
						],
						yAxisID: 'y-humidity'
					},
					{
						label: 'Temperature Ant',
						borderColor: '#ff7f0e',
						fill: false,
						showLine: true,
						lineTension: 0,
						data: [
							{x:0,y:20},
							{x:1,y:20.3},
							{x:2,y:19.8},
							{x:3,y:21.2},
							{x:4,y:20.8}
						],
						yAxisID: 'y-temperature'
					},
					{
						label: 'Temperature Aux 0',
						fill: false,
						borderColor: '#2ca02c',
						showLine: true,
						lineTension: 0,
						data: [
							{x:0,y:20.1},
							{x:1,y:20.2},
							{x:2,y:20.0},
							{x:3,y:21.3},
							{x:4,y:20.6}
						],
						yAxisID: 'y-temperature'
					},
					{
						label: 'Temperature Aux 1',
						borderColor: '#17becf',
						fill: false,
						showLine: true,
						lineTension: 0,
						data: [
							{x:0,y:21},
							{x:1,y:21.3},
							{x:2,y:20.8},
							{x:3,y:22.2},
							{x:4,y:21.8}
						],
						yAxisID: 'y-temperature'
					},
					{
						label: 'Temperature Aux 2',
						borderColor: '#ff6384',
						fill: false,
						showLine: true,
						lineTension: 0,
						data: [
							{x:0,y:20.5},
							{x:1,y:20.9},
							{x:2,y:20.3},
							{x:3,y:21.6},
							{x:4,y:21.2}
						],
						yAxisID: 'y-temperature'
					}
				],
			},
			options: {
				responsive: false,
				legend: {position: 'bottom'},
				scales: {
					xAxes: [
						{
							scaleLabel:{display: true,labelString: 'Time (m)'},
							display: true,
						}
					],
					yAxes:[
						{
							type: 'linear',
							display: true,
							position: 'right',
							id: 'y-humidity',
							gridLines: { drawOnChartArea: false},
							scaleLabel:{display: true,labelString: 'Humidity (%)'},
							ticks: {min: 20.0,max:85.0}
						},
						{
							type: 'linear',
							display: true,
							position: 'left',
							id: 'y-temperature',
							scaleLabel:{display: true,labelString: 'Temperature (°C)'},
							ticks: {min: 15.0,max:35.0}
						}
					]
				}
			}
		});

    }


	updateChart() {
		console.time('updateChart');
		let window = '';
		switch(this.timeWindow) {
			case TimeWindow.Hour:
				window = 'hour';
				break;
			case TimeWindow.Week:
				window = 'week';
				break
			case TimeWindow.Day:
				window = 'day';
				break;
			default:
				window = 'hour';
				break;
		}

		this.climateReport.getReport(this.hostName,this.zoneName,window).subscribe((cr) => {
			this.chart.data.datasets[0].data = [];
			this.chart.data.datasets[1].data = [];
			this.chart.data.datasets[2].data = [];
			this.chart.data.datasets[3].data = [];
			this.chart.data.datasets[4].data = [];
			for (let h of cr.Humidity) {
				this.chart.data.datasets[0].data.push({x:h.X,y:h.Y});
			}
			for (let t of cr.TemperatureAnt) {
				this.chart.data.datasets[1].data.push({x:t.X,y:t.Y});
			}
			for (let t of cr.TemperatureAux1) {
				this.chart.data.datasets[2].data.push({x:t.X,y:t.Y});
			}
			for (let t of cr.TemperatureAux2) {
				this.chart.data.datasets[3].data.push({x:t.X,y:t.Y});
			}
			for (let t of cr.TemperatureAux3) {
				this.chart.data.datasets[4].data.push({x:t.X,y:t.Y});
			}
			this.chart.update();
			console.timeEnd('updateChart');
		})

	}

	isSelected(w : TimeWindow) {
		if ( w == this.timeWindow ) {
			return ' active'
		}
		return ''
	}

	public selectTimeWindow(w: TimeWindow) {
		this.timeWindow = w;
		this.updateChart();
	}



}
