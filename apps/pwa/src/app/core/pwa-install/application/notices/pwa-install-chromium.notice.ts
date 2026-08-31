import { Injectable, inject } from '@angular/core';
import type { SystemNotice, SystemNoticeHandle } from '@bowerbird/system-notices';
import { DeclineAutoPromptCommand } from '../decline-auto-prompt.command';
import { EngagementEventHandler } from '../engagement-event.handler';
import { PwaInstallCoordinator } from '../pwa-install.coordinator';
import { PwaRuntimeService } from '../../infrastructure/pwa-runtime.service';
import { ViewportAdapter } from '../../infrastructure/viewport.adapter';
import { InstallPromptPresenter } from '../../presentation/install-prompt.presenter';

@Injectable()
export class PwaInstallChromiumNotice implements SystemNotice {
  readonly id = 'pwa-install-chromium';
  readonly priority = 50;
  readonly scope = 'tenant' as const;

  private readonly coordinator = inject(PwaInstallCoordinator);
  private readonly runtime = inject(PwaRuntimeService);
  private readonly viewport = inject(ViewportAdapter);
  private readonly presenter = inject(InstallPromptPresenter);
  private readonly decline = inject(DeclineAutoPromptCommand);
  private readonly events = inject(EngagementEventHandler);

  canShow(): boolean {
    return this.runtime.canInstallNative() && this.coordinator.canShowAutoPrompt();
  }

  show(handle: SystemNoticeHandle): void {
    const variant = this.viewport.isMobile() ? 'sheet' : 'snackbar';
    this.events.track('pwa_install_prompt_shown', { variant });

    const actions = {
      onInstall: () => this.handleInstall(handle),
      onNotNow: () => this.handleDismiss('not_now', handle),
      onTimeout: () => this.handleDismiss('timeout', handle),
      onContinueBrowser: () => this.handleDismiss('continue_browser', handle),
    };

    if (this.viewport.isMobile()) {
      this.presenter.openMobileSheet(actions);
      return;
    }

    this.presenter.openDesktopSnackbar(actions);
  }

  dismiss(reason: string, handle: SystemNoticeHandle): void {
    this.handleDismiss(reason, handle);
  }

  private handleInstall(handle: SystemNoticeHandle): void {
    this.events.track('pwa_install_prompt_action', { action: 'install' });
    void this.runtime.promptInstall().then((outcome) => {
      this.events.track('pwa_install_native_result', { outcome });
      if (outcome === 'accepted') {
        this.events.track('pwa_installed', {});
      }
      handle.clearActive(outcome);
    });
  }

  private handleDismiss(reason: string, handle: SystemNoticeHandle): void {
    this.events.track('pwa_install_prompt_action', { action: reason });
    this.decline.execute(reason);
    handle.clearActive(reason);
    handle.tryPresent(this.scope);
  }
}
