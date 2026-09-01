import { DatePipe } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { PartiesStore } from '../../../application/parties.store';
import { roleLabel, creationSourceLabel } from '../../../domain/party.model';

@Component({
  selector: 'app-parties-detail',
  standalone: true,
  imports: [DatePipe, RouterLink, NgIcon, HlmCardImports, HlmButtonImports, HlmBadgeImports, HlmSpinnerImports, HlmAlertImports],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-lg space-y-6">
      <a routerLink=".." class="inline-flex items-center text-sm text-muted-foreground hover:text-foreground">
        <ng-icon name="lucideArrowLeft" class="mr-1" />
        Volver a contactos
      </a>

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      @if (store.loading() && !store.selectedParty()) {
        <div class="flex justify-center py-16"><hlm-spinner class="size-8" /></div>
      }

      @if (store.selectedParty(); as party) {
        <header class="flex items-start justify-between gap-3">
          <div>
            <h1 class="text-2xl font-semibold tracking-tight">{{ party.name }}</h1>
            <p class="mt-1 text-sm text-muted-foreground">Detalle del contacto</p>
          </div>
          <a hlmBtn variant="outline" routerLink="edit">Editar</a>
        </header>

        <hlm-card class="space-y-4 p-6">
          <div class="grid gap-3 text-sm">
            <div>
              <p class="text-muted-foreground">NIT</p>
              <p class="font-medium">{{ party.tax_id || '—' }}</p>
            </div>
            <div>
              <p class="text-muted-foreground">Roles</p>
              <div class="flex flex-wrap gap-1">
                @for (role of party.roles; track role) {
                  <span hlmBadge variant="secondary">{{ roleLabel(role) }}</span>
                }
              </div>
            </div>
            <div>
              <p class="text-muted-foreground">Origen</p>
              <p class="font-medium">{{ creationSourceLabel(party.creation_source) }}</p>
            </div>
            <div>
              <p class="text-muted-foreground">Estado</p>
              <p class="font-medium">{{ party.status }}</p>
            </div>
            <div>
              <p class="text-muted-foreground">Creado</p>
              <p>{{ party.created_at | date: 'medium' }}</p>
            </div>
            <div>
              <p class="text-muted-foreground">Actualizado</p>
              <p>{{ party.updated_at | date: 'medium' }}</p>
            </div>
          </div>
        </hlm-card>
      }
    </div>
  `,
})
export class DetailPartyPage implements OnInit {
  readonly store = inject(PartiesStore);
  readonly roleLabel = roleLabel;
  readonly creationSourceLabel = creationSourceLabel;
  private readonly route = inject(ActivatedRoute);

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('partyId');
    if (id) this.store.loadParty(id);
  }
}
