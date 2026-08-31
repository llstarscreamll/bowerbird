import { InjectionToken } from '@angular/core';

export interface AnalyticsPort {
  track(event: string, properties?: Record<string, unknown>): void;
}

export const ANALYTICS_PORT = new InjectionToken<AnalyticsPort>('ANALYTICS_PORT');
