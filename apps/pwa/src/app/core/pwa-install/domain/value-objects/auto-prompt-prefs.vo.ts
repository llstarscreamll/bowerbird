import { DismissReason, type DismissReasonCode } from './dismiss-reason.vo';

const MS_PER_DAY = 24 * 60 * 60 * 1000;

export interface AutoPromptPrefsDto {
  dismissCount: number;
  lastDismissedAt: string | null;
  lastDismissReason: DismissReasonCode | null;
  permanentlySilenced: boolean;
}

export class AutoPromptPrefs {
  private constructor(
    readonly dismissCount: number,
    readonly lastDismissedAt: Date | null,
    readonly lastDismissReason: DismissReasonCode | null,
    readonly permanentlySilenced: boolean,
  ) {}

  static initial(): AutoPromptPrefs {
    return new AutoPromptPrefs(0, null, null, false);
  }

  static reconstitute(dto: AutoPromptPrefsDto): AutoPromptPrefs {
    return new AutoPromptPrefs(dto.dismissCount, dto.lastDismissedAt ? new Date(dto.lastDismissedAt) : null, dto.lastDismissReason, dto.permanentlySilenced);
  }

  decline(reason: DismissReason, now: Date): AutoPromptPrefs {
    const explicit = reason.isExplicit();
    const nextDismissCount = explicit ? this.dismissCount + 1 : this.dismissCount;
    const permanentlySilenced = nextDismissCount >= 3;

    return new AutoPromptPrefs(nextDismissCount, now, reason.code, permanentlySilenced);
  }

  isSilenced(): boolean {
    return this.permanentlySilenced;
  }

  isCooldownActive(now: Date): boolean {
    if (!this.lastDismissedAt || !this.lastDismissReason) {
      return false;
    }

    const days = this.cooldownDaysFor(this.lastDismissReason, this.dismissCount);
    const elapsed = now.getTime() - this.lastDismissedAt.getTime();
    return elapsed < days * MS_PER_DAY;
  }

  private cooldownDaysFor(reason: DismissReasonCode, dismissCount: number): number {
    if (dismissCount >= 2) {
      return 30;
    }

    if (reason === 'timeout') {
      return 3;
    }

    return 7;
  }

  toDto(): AutoPromptPrefsDto {
    return {
      dismissCount: this.dismissCount,
      lastDismissedAt: this.lastDismissedAt?.toISOString() ?? null,
      lastDismissReason: this.lastDismissReason,
      permanentlySilenced: this.permanentlySilenced,
    };
  }
}
