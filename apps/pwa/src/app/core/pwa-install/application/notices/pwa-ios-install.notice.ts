import { Injectable, inject } from '@angular/core';
import type { SystemNotice, SystemNoticeHandle } from '@bowerbird/system-notices';
import { DeclineAutoPromptCommand } from '../decline-auto-prompt.command';
import { EngagementEventHandler } from '../engagement-event.handler';
import { PwaInstallCoordinator } from '../pwa-install.coordinator';
import { PwaRuntimeService } from '../../infrastructure/pwa-runtime.service';
import { InstallPromptPresenter } from '../../presentation/install-prompt.presenter';

@Injectable()
export class PwaIosInstallNotice implements SystemNotice {
  readonly id = 'pwa-install-ios';
  readonly priority = 50;
  readonly scope = 'tenant' as const;

  private readonly coordinator = inject(PwaInstallCoordinator);
  private readonly runtime = inject(PwaRuntimeService);
  private readonly presenter = inject(InstallPromptPresenter);
  private readonly decline = inject(DeclineAutoPromptCommand);
  private readonly events = inject(EngagementEventHandler);

  canShow(): boolean {
    return this.runtime.canShowIosGuide() && this.coordinator.canShowAutoPrompt();
  }

  show(handle: SystemNoticeHandle): void {
    this.events.track('pwa_install_prompt_shown', { variant: 'ios_guide' });
    this.presenter.openIosSheet(() => this.handleDismiss('not_now', handle));
  }

  dismiss(reason: string, handle: SystemNoticeHandle): void {
    this.handleDismiss(reason, handle);
  }

  private handleDismiss(reason: string, handle: SystemNoticeHandle): void {
    this.events.track('pwa_install_prompt_action', { action: reason });
    this.decline.execute(reason);
    handle.clearActive(reason);
    handle.tryPresent(this.scope);
  }
}
