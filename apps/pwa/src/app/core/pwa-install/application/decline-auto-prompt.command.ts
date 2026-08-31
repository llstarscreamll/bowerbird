import { Injectable, inject } from '@angular/core';
import { DismissReason } from '../domain/value-objects/dismiss-reason.vo';
import { CLOCK_PORT } from './ports/clock.port';
import { INSTALL_ENGAGEMENT_REPOSITORY } from './ports/engagement.repository.port';
import { EngagementEventHandler } from './engagement-event.handler';

@Injectable({ providedIn: 'root' })
export class DeclineAutoPromptCommand {
  private readonly repository = inject(INSTALL_ENGAGEMENT_REPOSITORY);
  private readonly clock = inject(CLOCK_PORT);
  private readonly events = inject(EngagementEventHandler);

  execute(reasonCode: string): void {
    const reason = DismissReason.fromUserAction(reasonCode);
    const engagement = this.repository.load();
    const result = engagement.declineAutoPrompt(reason, this.clock.now());
    this.repository.save(result.engagement);
    this.events.dispatch(result.events);
  }
}
