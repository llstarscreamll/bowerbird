import { DOCUMENT, isPlatformBrowser } from '@angular/common';
import { Injectable, PLATFORM_ID, inject } from '@angular/core';
import { InstallEngagement } from '../domain/install-engagement.aggregate';
import { AutoPromptPrefs, type AutoPromptPrefsDto } from '../domain/value-objects/auto-prompt-prefs.vo';
import { VisitHistory } from '../domain/value-objects/visit-history.vo';
import type { InstallEngagementRepository } from '../application/ports/engagement.repository.port';
import { ENGAGEMENT_STORAGE_KEYS } from './engagement-storage.keys';

interface PersistedEngagement {
  visits: string[];
  prefs: AutoPromptPrefsDto;
}

@Injectable()
export class EngagementStorageRepository implements InstallEngagementRepository {
  private readonly document = inject(DOCUMENT);
  private readonly platformId = inject(PLATFORM_ID);

  load(): InstallEngagement {
    if (!isPlatformBrowser(this.platformId)) {
      return InstallEngagement.reconstitute(VisitHistory.empty(), AutoPromptPrefs.initial());
    }

    const storage = this.document.defaultView?.localStorage;
    if (!storage) {
      return InstallEngagement.reconstitute(VisitHistory.empty(), AutoPromptPrefs.initial());
    }

    const visitsRaw = storage.getItem(ENGAGEMENT_STORAGE_KEYS.visits);
    const prefsRaw = storage.getItem(ENGAGEMENT_STORAGE_KEYS.prefs);

    const visits = visitsRaw ? VisitHistory.reconstitute(JSON.parse(visitsRaw) as string[]) : VisitHistory.empty();
    const prefs = prefsRaw ? AutoPromptPrefs.reconstitute(JSON.parse(prefsRaw) as AutoPromptPrefsDto) : AutoPromptPrefs.initial();

    return InstallEngagement.reconstitute(visits, prefs);
  }

  save(engagement: InstallEngagement): void {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }

    const storage = this.document.defaultView?.localStorage;
    if (!storage) {
      return;
    }

    const payload: PersistedEngagement = {
      visits: engagement.visits.toISOStrings(),
      prefs: engagement.prefs.toDto(),
    };

    storage.setItem(ENGAGEMENT_STORAGE_KEYS.visits, JSON.stringify(payload.visits));
    storage.setItem(ENGAGEMENT_STORAGE_KEYS.prefs, JSON.stringify(payload.prefs));
  }

  hasActiveSession(): boolean {
    if (!isPlatformBrowser(this.platformId)) {
      return true;
    }

    const sessionStorage = this.document.defaultView?.sessionStorage;
    return sessionStorage?.getItem(ENGAGEMENT_STORAGE_KEYS.session) === '1';
  }

  markSessionActive(): void {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }

    this.document.defaultView?.sessionStorage?.setItem(ENGAGEMENT_STORAGE_KEYS.session, '1');
  }
}
