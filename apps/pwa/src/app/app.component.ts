import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { PwaService } from './core/services/pwa.service';
import { ToastContainerComponent } from './core/presentation/components/toast/toast.component';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, ToastContainerComponent],
  template: `
    <router-outlet></router-outlet>
    <app-toast-container></app-toast-container>

    @if (pwa.canInstall()) {
      <div class="fixed bottom-4 right-4 z-50 max-w-sm card">
        <div class="flex items-start">
          <div class="flex-shrink-0">
            <span class="material-icons-outlined text-indigo-600 dark:text-indigo-400">system_update</span>
          </div>
          <div class="ml-3 w-0 flex-1 pt-0.5">
            <p class="text-sm font-semibold text-slate-900 dark:text-white">Install Bowerbird</p>
            <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">Install the app for faster launch and offline access.</p>
            <div class="mt-3 flex space-x-3">
              <button type="button" class="btn-primary py-1.5 px-3 text-xs" (click)="install()">Install app</button>
            </div>
          </div>
        </div>
      </div>
    }

    @if (pwa.updateAvailable()) {
      <div class="fixed bottom-4 left-4 z-50 max-w-sm card border-indigo-200 dark:border-indigo-500/30 bg-indigo-50/50 dark:bg-indigo-900/10">
        <div class="flex items-start">
          <div class="flex-shrink-0">
            <span class="material-icons-outlined text-indigo-600 dark:text-indigo-400">tips_and_updates</span>
          </div>
          <div class="ml-3 w-0 flex-1 pt-0.5">
            <p class="text-sm font-semibold text-indigo-900 dark:text-indigo-200">Update available</p>
            <p class="mt-1 text-sm text-indigo-700 dark:text-indigo-300">A new version is ready. Refresh to apply the update.</p>
            <div class="mt-3 flex space-x-3">
              <button type="button" class="btn-primary py-1.5 px-3 text-xs" (click)="refreshWithUpdate()">Refresh now</button>
            </div>
          </div>
        </div>
      </div>
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppComponent {
  readonly pwa = inject(PwaService);

  install(): void {
    void this.pwa.promptInstall();
  }

  refreshWithUpdate(): void {
    void this.pwa.activateUpdateAndReload();
  }
}
