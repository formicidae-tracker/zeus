import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';

import { AppComponent } from './app.component';
import { NgbModule } from '@ng-bootstrap/ng-bootstrap';

import { RouterModule } from '@angular/router';
import { ROUTES } from './app.route';

import { HomeComponent } from './home/home.component';
import { ZoneComponent } from './zone/zone.component';
import { PageNotFoundComponent } from './page-not-found/page-not-found.component';
import { ZonePreviewComponent } from './zone-preview/zone-preview.component';

import { ZoneService } from './zone.service';
import { ClimateChartComponent } from './climate-chart/climate-chart.component';
import { AlarmComponent } from './alarm/alarm.component';

@NgModule({
    imports: [
  		NgbModule,
		BrowserModule,
        RouterModule.forRoot(ROUTES)
	],
	declarations: [
		AppComponent,
		HomeComponent,
		ZoneComponent,
		PageNotFoundComponent,
		ZonePreviewComponent,
		ClimateChartComponent,
		AlarmComponent
	],
	providers: [ZoneService],
	bootstrap: [AppComponent]
})
export class AppModule { }
