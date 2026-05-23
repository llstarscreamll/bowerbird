import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { PwaService } from './core/services/pwa.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet],
  template: `
    <router-outlet></router-outlet>

    @if (pwa.canInstall()) {
      <div class="fixed bottom-4 right-4 z-50 max-w-sm rounded-xl border border-cyan-200 bg-white p-4 shadow-lg">
        <p class="text-sm font-semibold text-slate-900">Install Bowerbird</p>
        <p class="mt-1 text-xs text-slate-600">Install the app for faster launch and offline access.</p>
        <button
          type="button"
          class="mt-3 rounded-md bg-cyan-700 px-3 py-2 text-sm font-semibold text-white hover:bg-cyan-800"
          (click)="install()"
        >
          Install app
        </button>
      </div>
    }

    @if (pwa.updateAvailable()) {
      <div class="fixed bottom-4 left-4 z-50 max-w-sm rounded-xl border border-emerald-200 bg-white p-4 shadow-lg">
        <p class="text-sm font-semibold text-slate-900">Update available</p>
        <p class="mt-1 text-xs text-slate-600">A new version is ready. Refresh to apply the update.</p>
        <button
          type="button"
          class="mt-3 rounded-md bg-emerald-700 px-3 py-2 text-sm font-semibold text-white hover:bg-emerald-800"
          (click)="refreshWithUpdate()"
        >
          Refresh now
        </button>
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
