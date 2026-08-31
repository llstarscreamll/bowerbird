import type { InstallEngagementDomainEvent } from './events/install-engagement.events';
import { AutoPromptPrefs } from './value-objects/auto-prompt-prefs.vo';
import type { DismissReason } from './value-objects/dismiss-reason.vo';
import { VisitHistory } from './value-objects/visit-history.vo';

export interface InstallEngagementMutation {
  engagement: InstallEngagement;
  events: InstallEngagementDomainEvent[];
}

export class InstallEngagement {
  private constructor(
    readonly visits: VisitHistory,
    readonly prefs: AutoPromptPrefs,
    private readonly wasEligible: boolean,
  ) {}

  static reconstitute(visits: VisitHistory, prefs: AutoPromptPrefs): InstallEngagement {
    const now = new Date();
    return new InstallEngagement(visits, prefs, visits.hasMetEngagementThreshold(now));
  }

  recordSessionVisit(now: Date): InstallEngagementMutation {
    const nextVisits = this.visits.recordVisit(now);
    const next = new InstallEngagement(nextVisits, this.prefs, this.wasEligible);
    const events: InstallEngagementDomainEvent[] = [{ type: 'SessionVisitRecorded', visitNumber: nextVisits.visitCount() }];

    if (!this.wasEligible && nextVisits.hasMetEngagementThreshold(now)) {
      events.push({ type: 'AutoPromptBecameEligible', visitNumber: nextVisits.visitCount() });
      return {
        engagement: new InstallEngagement(nextVisits, this.prefs, true),
        events,
      };
    }

    return { engagement: next, events };
  }

  declineAutoPrompt(reason: DismissReason, now: Date): InstallEngagementMutation {
    const nextPrefs = this.prefs.decline(reason, now);
    return {
      engagement: new InstallEngagement(this.visits, nextPrefs, this.wasEligible),
      events: [{ type: 'AutoPromptDeclined', reason: reason.code }],
    };
  }

  isEligibleForAutoPrompt(now: Date): boolean {
    return this.visits.hasMetEngagementThreshold(now);
  }

  canShowAutoPrompt(now: Date): boolean {
    return this.isEligibleForAutoPrompt(now) && !this.prefs.isSilenced() && !this.prefs.isCooldownActive(now);
  }
}
