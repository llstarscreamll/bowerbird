import { ApplicationRef, DestroyRef, Injectable, PLATFORM_ID, inject, isDevMode, signal } from '@angular/core';
import { DOCUMENT, isPlatformBrowser } from '@angular/common';
import { SwUpdate, VersionReadyEvent } from '@angular/service-worker';
import { filter, first } from 'rxjs';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed'; platform: string }>;
}

@Injectable({ providedIn: 'root' })
export class PwaService {
  private readonly document = inject(DOCUMENT);
  private readonly platformId = inject(PLATFORM_ID);
  private readonly appRef = inject(ApplicationRef);
  private readonly destroyRef = inject(DestroyRef);
  private readonly swUpdate = inject(SwUpdate, { optional: true });

  private deferredInstallPrompt: BeforeInstallPromptEvent | null = null;

  readonly canInstall = signal(false);
  readonly updateAvailable = signal(false);

  constructor() {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }

    this.setupInstallPromptCapture();
    this.setupServiceWorkerUpdates();
  }

  async promptInstall(): Promise<void> {
    if (!this.deferredInstallPrompt) {
      return;
    }

    const installEvent = this.deferredInstallPrompt;
    this.deferredInstallPrompt = null;
    this.canInstall.set(false);

    await installEvent.prompt();
    await installEvent.userChoice;
  }

  async activateUpdateAndReload(): Promise<void> {
    if (!this.swUpdate?.isEnabled) {
      return;
    }

    await this.swUpdate.activateUpdate();
    this.document.location.reload();
  }

  private setupInstallPromptCapture(): void {
    this.document.defaultView?.addEventListener('beforeinstallprompt', (event: Event) => {
      event.preventDefault();
      this.deferredInstallPrompt = event as BeforeInstallPromptEvent;
      this.canInstall.set(true);
    });

    this.document.defaultView?.addEventListener('appinstalled', () => {
      this.deferredInstallPrompt = null;
      this.canInstall.set(false);
    });
  }

  private setupServiceWorkerUpdates(): void {
    if (!this.swUpdate?.isEnabled || isDevMode()) {
      return;
    }

    const stable$ = this.appRef.isStable.pipe(first((isStable) => isStable));

    const stableSubscription = stable$.subscribe(() => {
      void this.swUpdate?.checkForUpdate();
      const intervalId = this.document.defaultView?.setInterval(
        () => {
          void this.swUpdate?.checkForUpdate();
        },
        6 * 60 * 60 * 1000,
      );

      this.destroyRef.onDestroy(() => {
        if (intervalId) {
          this.document.defaultView?.clearInterval(intervalId);
        }
      });
    });

    this.destroyRef.onDestroy(() => stableSubscription.unsubscribe());

    const updatesSubscription = this.swUpdate.versionUpdates
      .pipe(filter((event): event is VersionReadyEvent => event.type === 'VERSION_READY'))
      .subscribe(() => this.updateAvailable.set(true));

    this.destroyRef.onDestroy(() => updatesSubscription.unsubscribe());
  }
}
