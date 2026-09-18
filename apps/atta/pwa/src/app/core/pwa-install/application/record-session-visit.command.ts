import { Injectable, inject } from '@angular/core';
import { CLOCK_PORT } from './ports/clock.port';
import { INSTALL_ENGAGEMENT_REPOSITORY } from './ports/engagement.repository.port';
import { EngagementStorageRepository } from '../infrastructure/engagement-storage.repository';

@Injectable({ providedIn: 'root' })
export class RecordSessionVisitCommand {
  private readonly repository = inject(INSTALL_ENGAGEMENT_REPOSITORY);
  private readonly storage = inject(EngagementStorageRepository);
  private readonly clock = inject(CLOCK_PORT);

  execute(): void {
    if (this.storage.hasActiveSession()) {
      return;
    }

    const now = this.clock.now();
    const engagement = this.repository.load();
    const next = engagement.recordSessionVisit(now);
    this.repository.save(next);
    this.storage.markSessionActive();
  }
}
