import { Component, OnInit, inject } from '@angular/core';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { LegalEntitiesStore } from '../../../legal-entities/application/legal-entities.store';
import { LegalEntityFormComponent, LegalEntityFormValue } from '../../../legal-entities/presentation/components/legal-entity-form.component';

@Component({
  selector: 'app-settings-page',
  standalone: true,
  imports: [NgIcon, HlmCardImports, HlmAlertImports, HlmSpinnerImports, LegalEntityFormComponent],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full' },
  template: `
    <div class="h-full w-full flex-1 overflow-y-auto p-8">
      <div class="mx-auto w-full max-w-lg flex flex-col gap-6">
        <header>
          <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">Configuración</h1>
          <p class="mt-1 text-sm text-muted-foreground">Identificación tributaria de esta organización. Se usa para reconocer las facturas que te envían.</p>
        </header>

        @if (store.errorMessage(); as err) {
          <div hlmAlert variant="destructive">
            <ng-icon name="lucideCircleAlert" hlmAlertIcon />
            <h4 hlmAlertTitle>Error</h4>
            <p hlmAlertDescription>{{ err }}</p>
          </div>
        }

        @if (store.loading()) {
          <div class="flex items-center justify-center py-16">
            <hlm-spinner class="size-8 text-primary" />
          </div>
        } @else {
          <hlm-card class="p-6">
            <h2 class="text-sm font-semibold">Razón social propia</h2>
            <p class="mt-1 mb-4 text-xs text-muted-foreground">Registra el NIT o la cédula con el que figuran las facturas electrónicas recibidas.</p>
            <app-legal-entity-form
              [initial]="store.current()"
              [submitting]="store.submitting()"
              [submitLabel]="store.hasAny() ? 'Guardar cambios' : 'Registrar identificación'"
              (submitted)="onSubmit($event)"
            />
          </hlm-card>
        }
      </div>
    </div>
  `,
})
export class SettingsPage implements OnInit {
  readonly store = inject(LegalEntitiesStore);

  ngOnInit(): void {
    this.store.load();
  }

  onSubmit(value: LegalEntityFormValue): void {
    const current = this.store.current();
    if (current) {
      this.store.update(current.id, value).subscribe();
      return;
    }
    this.store.create(value).subscribe();
  }
}
