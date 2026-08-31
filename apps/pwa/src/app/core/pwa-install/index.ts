import { EnvironmentProviders, makeEnvironmentProviders } from '@angular/core';
import { SYSTEM_NOTICE } from '@bowerbird/system-notices/angular';
import { CLOCK_PORT } from './application/ports/clock.port';
import { INSTALL_ENGAGEMENT_REPOSITORY } from './application/ports/engagement.repository.port';
import { PwaInstallChromiumNotice } from './application/notices/pwa-install-chromium.notice';
import { PwaIosInstallNotice } from './application/notices/pwa-ios-install.notice';
import { PwaUpdateNotice } from './application/notices/pwa-update.notice';
import { EngagementStorageRepository } from './infrastructure/engagement-storage.repository';
import { SystemClockAdapter } from './infrastructure/system-clock.adapter';

export function providePwaInstall(): EnvironmentProviders {
  return makeEnvironmentProviders([
    EngagementStorageRepository,
    { provide: INSTALL_ENGAGEMENT_REPOSITORY, useExisting: EngagementStorageRepository },
    { provide: CLOCK_PORT, useClass: SystemClockAdapter },
    { provide: SYSTEM_NOTICE, useClass: PwaUpdateNotice, multi: true },
    { provide: SYSTEM_NOTICE, useClass: PwaInstallChromiumNotice, multi: true },
    { provide: SYSTEM_NOTICE, useClass: PwaIosInstallNotice, multi: true },
  ]);
}

export { PwaInstallCoordinator } from './application/pwa-install.coordinator';
export { InstallPromptHostComponent } from './presentation/install-prompt-host.component';
