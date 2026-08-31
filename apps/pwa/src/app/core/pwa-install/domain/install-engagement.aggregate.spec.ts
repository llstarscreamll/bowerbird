import { describe, expect, it } from 'vitest';
import { InstallEngagement } from './install-engagement.aggregate';
import { AutoPromptPrefs } from './value-objects/auto-prompt-prefs.vo';
import { DismissReason } from './value-objects/dismiss-reason.vo';
import { VisitHistory } from './value-objects/visit-history.vo';

describe('InstallEngagement', () => {
  const day1 = new Date('2026-01-01T10:00:00Z');
  const day3 = new Date('2026-01-03T10:00:00Z');
  const day10 = new Date('2026-01-10T10:00:00Z');

  it('does not allow auto prompt on first visit', () => {
    const engagement = InstallEngagement.reconstitute(VisitHistory.empty(), AutoPromptPrefs.initial());
    const result = engagement.recordSessionVisit(day1);

    expect(result.engagement.isEligibleForAutoPrompt(day1)).toBe(false);
    expect(result.engagement.canShowAutoPrompt(day1)).toBe(false);
  });

  it('allows auto prompt on second visit within 7 days', () => {
    let engagement = InstallEngagement.reconstitute(VisitHistory.empty(), AutoPromptPrefs.initial());
    engagement = engagement.recordSessionVisit(day1).engagement;
    const result = engagement.recordSessionVisit(day3);

    expect(result.engagement.isEligibleForAutoPrompt(day3)).toBe(true);
    expect(result.engagement.canShowAutoPrompt(day3)).toBe(true);
    expect(result.events.some((event) => event.type === 'AutoPromptBecameEligible')).toBe(true);
  });

  it('does not allow auto prompt when second visit is after 7 days', () => {
    let engagement = InstallEngagement.reconstitute(VisitHistory.empty(), AutoPromptPrefs.initial());
    engagement = engagement.recordSessionVisit(day1).engagement;
    const result = engagement.recordSessionVisit(day10);

    expect(result.engagement.isEligibleForAutoPrompt(day10)).toBe(false);
  });

  it('applies atomic decline with cooldown', () => {
    let engagement = InstallEngagement.reconstitute(VisitHistory.reconstitute([day1.toISOString(), day3.toISOString()]), AutoPromptPrefs.initial());
    engagement = engagement.recordSessionVisit(day3).engagement;
    const declined = engagement.declineAutoPrompt(DismissReason.fromUserAction('not_now'), day3);

    expect(declined.engagement.canShowAutoPrompt(day3)).toBe(false);
    expect(declined.engagement.prefs.dismissCount).toBe(1);
    expect(declined.engagement.prefs.lastDismissReason).toBe('not_now');
  });

  it('silences after third explicit dismiss', () => {
    let prefs = AutoPromptPrefs.initial();
    const visits = VisitHistory.reconstitute([day1.toISOString(), day3.toISOString()]);
    let engagement = InstallEngagement.reconstitute(visits, prefs);

    for (let i = 0; i < 3; i++) {
      const result = engagement.declineAutoPrompt(DismissReason.fromUserAction('not_now'), day3);
      engagement = result.engagement;
      prefs = engagement.prefs;
    }

    expect(engagement.prefs.isSilenced()).toBe(true);
    expect(engagement.canShowAutoPrompt(day10)).toBe(false);
  });
});
