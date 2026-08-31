import { Injectable, signal } from '@angular/core';
import { toast } from '@spartan-ng/brain/sonner';

export interface InstallPromptActions {
  onInstall: () => void;
  onNotNow: () => void;
  onTimeout?: () => void;
  onContinueBrowser?: () => void;
}

@Injectable({ providedIn: 'root' })
export class InstallPromptPresenter {
  readonly mobileSheetOpen = signal(false);
  readonly iosSheetOpen = signal(false);

  private mobileActions: InstallPromptActions | null = null;

  openDesktopSnackbar(actions: InstallPromptActions): void {
    toast('Instala Bowerbird', {
      description: 'Tu espacio de trabajo, a un toque.',
      duration: 6000,
      action: {
        label: 'Instalar',
        onClick: actions.onInstall,
      },
      cancel: {
        label: 'Ahora no',
        onClick: actions.onNotNow,
      },
      onDismiss: () => (actions.onTimeout ?? actions.onNotNow)(),
    });
  }

  openMobileSheet(actions: InstallPromptActions): void {
    this.mobileActions = actions;
    this.mobileSheetOpen.set(true);
  }

  closeMobileSheet(): void {
    this.mobileSheetOpen.set(false);
    this.mobileActions = null;
  }

  getMobileActions(): InstallPromptActions | null {
    return this.mobileActions;
  }

  openIosSheet(onClose: () => void): void {
    this.iosCloseHandler = onClose;
    this.iosSheetOpen.set(true);
  }

  closeIosSheet(): void {
    this.iosSheetOpen.set(false);
    this.iosCloseHandler?.();
    this.iosCloseHandler = null;
  }

  private iosCloseHandler: (() => void) | null = null;
}
