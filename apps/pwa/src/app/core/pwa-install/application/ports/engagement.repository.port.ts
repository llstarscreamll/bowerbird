import { InjectionToken } from '@angular/core';
import type { InstallEngagement } from '../../domain/install-engagement.aggregate';

export interface InstallEngagementRepository {
  load(): InstallEngagement;
  save(engagement: InstallEngagement): void;
}

export const INSTALL_ENGAGEMENT_REPOSITORY = new InjectionToken<InstallEngagementRepository>('INSTALL_ENGAGEMENT_REPOSITORY');
