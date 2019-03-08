import { Routes } from '@angular/router';

import { HomeComponent } from './home/home.component';
import { ZoneComponent } from './zone/zone.component';
import { PageNotFoundComponent } from './page-not-found/page-not-found.component';

export const ROUTES: Routes = [
    { path: '', component: HomeComponent },
    { path: 'host/:hostName/zone/:zoneName', component: ZoneComponent },
    { path: 'not-found', component: PageNotFoundComponent},
    { path: '**', redirectTo: '/not-found'}
];
