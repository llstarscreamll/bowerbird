import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmToaster } from '@spartan-ng/helm/sonner';
import { PwaService } from './core/services/pwa.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, HlmToaster, HlmCardImports, HlmButtonImports, NgIcon],
  template: `
    <router-outlet></router-outlet>
    <hlm-toaster richColors closeButton position="bottom-right" />

    @if (pwa.canInstall()) {
      <hlm-card class="fixed bottom-4 right-4 z-50 min-w-[300px] max-w-sm p-4">
        <div class="flex items-start gap-3">
          <ng-icon name="lucideDownload" class="text-primary shrink-0 text-xl" />
          <div class="flex-1 pt-0.5">
            <p class="text-sm font-semibold">Instalar Bowerbird</p>
            <p class="mt-1 text-sm text-muted-foreground">Instala la aplicación para un acceso más rápido.</p>
            <button type="button" hlmBtn size="sm" class="mt-3" (click)="install()">Instalar aplicación</button>
          </div>
        </div>
      </hlm-card>
    }

    @if (pwa.updateAvailable()) {
      <hlm-card class="fixed bottom-4 left-4 z-50 min-w-[300px] max-w-sm border-primary/30 bg-primary/5 p-4">
        <div class="flex items-start gap-3">
          <ng-icon name="lucideLightbulb" class="text-primary shrink-0 text-xl" />
          <div class="flex-1 pt-0.5">
            <p class="text-sm font-semibold text-primary">Update available</p>
            <p class="mt-1 text-sm text-muted-foreground">A new version is ready. Refresh to apply the update.</p>
            <button type="button" hlmBtn size="sm" class="mt-3" (click)="refreshWithUpdate()">Refresh now</button>
          </div>
        </div>
      </hlm-card>
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
