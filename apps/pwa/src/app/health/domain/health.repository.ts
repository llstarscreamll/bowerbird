import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';
import { HealthInfo } from './health.model';

export interface HealthRepository {
  checkHealth(): Observable<HealthInfo>;
}

export const HEALTH_REPOSITORY = new InjectionToken<HealthRepository>('HEALTH_REPOSITORY');
