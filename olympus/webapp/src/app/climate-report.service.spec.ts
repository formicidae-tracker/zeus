import { TestBed } from '@angular/core/testing';

import { ClimateReportService } from './climate-report.service';

describe('ClimateReportService', () => {
  beforeEach(() => TestBed.configureTestingModule({}));

  it('should be created', () => {
    const service: ClimateReportService = TestBed.get(ClimateReportService);
    expect(service).toBeTruthy();
  });
});
