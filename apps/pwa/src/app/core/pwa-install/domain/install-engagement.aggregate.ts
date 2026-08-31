import { AutoPromptPrefs } from './value-objects/auto-prompt-prefs.vo';
import type { DismissReason } from './value-objects/dismiss-reason.vo';
import { VisitHistory } from './value-objects/visit-history.vo';

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

  recordSessionVisit(now: Date): InstallEngagement {
    const nextVisits = this.visits.recordVisit(now);

    if (!this.wasEligible && nextVisits.hasMetEngagementThreshold(now)) {
      return new InstallEngagement(nextVisits, this.prefs, true);
    }

    return new InstallEngagement(nextVisits, this.prefs, this.wasEligible);
  }

  declineAutoPrompt(reason: DismissReason, now: Date): InstallEngagement {
    const nextPrefs = this.prefs.decline(reason, now);
    return new InstallEngagement(this.visits, nextPrefs, this.wasEligible);
  }

  isEligibleForAutoPrompt(now: Date): boolean {
    return this.visits.hasMetEngagementThreshold(now);
  }

  canShowAutoPrompt(now: Date): boolean {
    return this.isEligibleForAutoPrompt(now) && !this.prefs.isSilenced() && !this.prefs.isCooldownActive(now);
  }
}
