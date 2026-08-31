import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmSheetImports } from '@spartan-ng/helm/sheet';
import { InstallPromptPresenter } from './install-prompt.presenter';

@Component({
  selector: 'app-install-prompt-host',
  standalone: true,
  imports: [HlmSheetImports, HlmButtonImports],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <hlm-sheet side="bottom" [state]="presenter.mobileSheetOpen() ? 'open' : 'closed'" (stateChanged)="onMobileState($event)">
      <hlm-sheet-content *hlmSheetPortal="let ctx" class="p-6">
        <h2 class="text-lg font-semibold">Instala Bowerbird</h2>
        <p class="mt-2 text-sm text-muted-foreground">Tu espacio de trabajo, a un toque.</p>
        <div class="mt-6 flex flex-col gap-2">
          <button type="button" hlmBtn (click)="install()">Instalar</button>
          <button type="button" hlmBtn variant="outline" (click)="continueBrowser()">Continuar en navegador</button>
          <button type="button" hlmBtn variant="ghost" (click)="notNow()">Ahora no</button>
        </div>
      </hlm-sheet-content>
    </hlm-sheet>

    <hlm-sheet side="bottom" [state]="presenter.iosSheetOpen() ? 'open' : 'closed'" (stateChanged)="onIosState($event)">
      <hlm-sheet-content *hlmSheetPortal="let ctx" class="p-6">
        <h2 class="text-lg font-semibold">Añade Bowerbird a tu inicio</h2>
        <ol class="mt-3 list-decimal space-y-2 pl-5 text-sm text-muted-foreground">
          <li>Toca el botón Compartir</li>
          <li>Selecciona «Añadir a pantalla de inicio»</li>
          <li>Confirma con «Añadir»</li>
        </ol>
        <button type="button" hlmBtn class="mt-6 w-full" (click)="iosClose()">Entendido</button>
      </hlm-sheet-content>
    </hlm-sheet>
  `,
})
export class InstallPromptHostComponent {
  readonly presenter = inject(InstallPromptPresenter);

  onMobileState(state: 'open' | 'closed'): void {
    if (state === 'closed') {
      this.presenter.closeMobileSheet();
    }
  }

  onIosState(state: 'open' | 'closed'): void {
    if (state === 'closed') {
      this.presenter.closeIosSheet();
    }
  }

  install(): void {
    this.presenter.getMobileActions()?.onInstall();
    this.presenter.closeMobileSheet();
  }

  continueBrowser(): void {
    this.presenter.getMobileActions()?.onContinueBrowser?.();
    this.presenter.closeMobileSheet();
  }

  notNow(): void {
    this.presenter.getMobileActions()?.onNotNow();
    this.presenter.closeMobileSheet();
  }

  iosClose(): void {
    this.presenter.closeIosSheet();
  }
}
