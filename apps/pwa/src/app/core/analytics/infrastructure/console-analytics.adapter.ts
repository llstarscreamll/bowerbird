import { Injectable, isDevMode } from '@angular/core';
import type { AnalyticsPort } from '../application/ports/analytics.port';

@Injectable({ providedIn: 'root' })
export class ConsoleAnalyticsAdapter implements AnalyticsPort {
  track(event: string, properties?: Record<string, unknown>): void {
    try {
      if (isDevMode()) {
        console.debug('[analytics]', event, properties);
      }
    } catch {
      // Never propagate analytics failures.
    }
  }
}
