import { Injectable, inject } from '@angular/core';
import { toast } from '@spartan-ng/brain/sonner';
import type { SystemNotice, SystemNoticeHandle } from '@bowerbird/system-notices';
import { PwaRuntimeService } from '../../infrastructure/pwa-runtime.service';

@Injectable()
export class PwaUpdateNotice implements SystemNotice {
  readonly id = 'pwa-update';
  readonly priority = 100;
  readonly scope = 'global' as const;

  private readonly runtime = inject(PwaRuntimeService);

  canShow(): boolean {
    return this.runtime.updateAvailable();
  }

  show(handle: SystemNoticeHandle): void {
    toast('Update available', {
      description: 'A new version is ready. Refresh to apply the update.',
      duration: 10000,
      action: {
        label: 'Refresh now',
        onClick: () => {
          void this.runtime.activateUpdateAndReload();
          handle.clearActive('accepted');
        },
      },
    });
  }

  dismiss(reason: string, handle: SystemNoticeHandle): void {
    handle.clearActive(reason);
  }
}
