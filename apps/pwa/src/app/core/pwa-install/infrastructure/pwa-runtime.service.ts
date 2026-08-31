import { DOCUMENT, isPlatformBrowser } from '@angular/common';
import { Injectable, PLATFORM_ID, inject, signal } from '@angular/core';
import { ApplicationRef, DestroyRef } from '@angular/core';
import { SwUpdate, VersionReadyEvent } from '@angular/service-worker';
import { filter, first } from 'rxjs';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed'; platform: string }>;
}

@Injectable({ providedIn: 'root' })
export class PwaRuntimeService {
  private readonly document = inject(DOCUMENT);
  private readonly platformId = inject(PLATFORM_ID);
  private readonly appRef = inject(ApplicationRef);
  private readonly destroyRef = inject(DestroyRef);
  private readonly swUpdate = inject(SwUpdate, { optional: true });

  private deferredInstallPrompt: BeforeInstallPromptEvent | null = null;

  readonly nativeInstallAvailable = signal(false);
  readonly updateAvailable = signal(false);

  constructor() {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }

    this.setupInstallPromptCapture();
    this.setupServiceWorkerUpdates();
  }

  isStandalone(): boolean {
    if (!isPlatformBrowser(this.platformId)) {
      return false;
    }

    return this.document.defaultView?.matchMedia('(display-mode: standalone)').matches ?? false;
  }

  canInstallNative(): boolean {
    return this.nativeInstallAvailable() && !this.isStandalone();
  }

  isIosSafari(): boolean {
    if (!isPlatformBrowser(this.platformId)) {
      return false;
    }

    const nav = this.document.defaultView?.navigator;
    if (!nav) {
      return false;
    }

    const ua = nav.userAgent;
    const isIosDevice = /iPad|iPhone|iPod/.test(ua) || (nav.platform === 'MacIntel' && nav.maxTouchPoints > 1);
    return isIosDevice && !this.canInstallNative();
  }

  canShowIosGuide(): boolean {
    return this.isIosSafari() && !this.isStandalone();
  }

  async promptInstall(): Promise<'accepted' | 'dismissed' | 'unavailable'> {
    if (!this.deferredInstallPrompt) {
      return 'unavailable';
    }

    const installEvent = this.deferredInstallPrompt;
    this.deferredInstallPrompt = null;
    this.nativeInstallAvailable.set(false);

    await installEvent.prompt();
    const choice = await installEvent.userChoice;
    return choice.outcome;
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
      this.nativeInstallAvailable.set(true);
    });

    this.document.defaultView?.addEventListener('appinstalled', () => {
      this.deferredInstallPrompt = null;
      this.nativeInstallAvailable.set(false);
    });
  }

  private setupServiceWorkerUpdates(): void {
    if (!this.swUpdate?.isEnabled) {
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

    const updatesSubscription = this.swUpdate.versionUpdates.pipe(filter((event): event is VersionReadyEvent => event.type === 'VERSION_READY')).subscribe(() => this.updateAvailable.set(true));

    this.destroyRef.onDestroy(() => updatesSubscription.unsubscribe());
  }
}
