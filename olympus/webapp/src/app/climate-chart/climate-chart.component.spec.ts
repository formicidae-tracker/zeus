import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { ClimateChartComponent } from './climate-chart.component';

describe('ClimateChartComponent', () => {
  let component: ClimateChartComponent;
  let fixture: ComponentFixture<ClimateChartComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ ClimateChartComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ClimateChartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
