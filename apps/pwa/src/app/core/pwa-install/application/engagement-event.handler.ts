import { Injectable, inject } from '@angular/core';
import { ANALYTICS_PORT } from '../../analytics';
import type { InstallEngagementDomainEvent } from '../domain/events/install-engagement.events';
import type { PwaAnalyticsEvent } from './pwa-analytics.events';

@Injectable({ providedIn: 'root' })
export class EngagementEventHandler {
  private readonly analytics = inject(ANALYTICS_PORT);

  dispatch(events: InstallEngagementDomainEvent[]): void {
    for (const event of events) {
      switch (event.type) {
        case 'SessionVisitRecorded':
          this.track('pwa_visit_recorded', { visitNumber: event.visitNumber });
          break;
        case 'AutoPromptBecameEligible':
          this.track('pwa_install_eligible', { visitNumber: event.visitNumber });
          break;
        case 'AutoPromptDeclined':
          this.track('pwa_install_prompt_action', { action: event.reason });
          break;
      }
    }
  }

  track(event: PwaAnalyticsEvent, properties?: Record<string, unknown>): void {
    this.analytics.track(event, properties);
  }
}
