import { Injectable, Injector, computed, effect, inject, isDevMode, untracked } from '@angular/core';
import { SystemNoticesOrchestratorService } from '@bowerbird/system-notices/angular';
import { CLOCK_PORT } from './ports/clock.port';
import { INSTALL_ENGAGEMENT_REPOSITORY } from './ports/engagement.repository.port';
import { PwaRuntimeService } from '../infrastructure/pwa-runtime.service';
import { RecordSessionVisitCommand } from './record-session-visit.command';
import { InstallPromptPresenter } from '../presentation/install-prompt.presenter';

@Injectable({ providedIn: 'root' })
export class PwaInstallCoordinator {
  private readonly repository = inject(INSTALL_ENGAGEMENT_REPOSITORY);
  private readonly runtime = inject(PwaRuntimeService);
  private readonly clock = inject(CLOCK_PORT);
  private readonly recordSessionVisit = inject(RecordSessionVisitCommand);
  private readonly presenter = inject(InstallPromptPresenter);
  private readonly injector = inject(Injector);

  readonly showMenuItem = computed(() => !this.runtime.isStandalone() && (this.runtime.canInstallNative() || this.runtime.canShowIosGuide()));

  constructor() {
    this.recordSessionVisit.execute();

    effect(() => {
      if (isDevMode()) {
        return;
      }

      if (this.runtime.nativeInstallAvailable()) {
        untracked(() => this.injector.get(SystemNoticesOrchestratorService).tryPresent('tenant'));
      }
    });
  }

  canShowAutoPrompt(): boolean {
    if (isDevMode() || this.runtime.isStandalone()) {
      return false;
    }

    const engagement = this.repository.load();
    const platformReady = this.runtime.canInstallNative() || this.runtime.canShowIosGuide();
    return platformReady && engagement.canShowAutoPrompt(this.clock.now());
  }

  async openFromMenu(): Promise<void> {
    if (this.runtime.canInstallNative()) {
      await this.runtime.promptInstall();
      return;
    }

    if (this.runtime.canShowIosGuide()) {
      this.presenter.openIosSheet(() => undefined);
    }
  }
}
