import { DatePipe } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { CatalogStore } from '../../../application/catalog.store';

@Component({
  selector: 'app-catalog-item-detail',
  standalone: true,
  imports: [DatePipe, RouterLink, NgIcon, HlmCardImports, HlmButtonImports, HlmBadgeImports, HlmSpinnerImports, HlmAlertImports],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-lg space-y-6">
      <a routerLink=".." class="inline-flex items-center text-sm text-muted-foreground hover:text-foreground">
        <ng-icon name="lucideArrowLeft" class="mr-1" />
        Volver al catálogo
      </a>

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      @if (store.loading() && !store.selectedItem()) {
        <div class="flex justify-center py-16"><hlm-spinner class="size-8" /></div>
      }

      @if (store.selectedItem(); as item) {
        <header class="flex items-start justify-between gap-3">
          <div>
            <h1 class="text-2xl font-semibold tracking-tight">{{ item.name }}</h1>
            <p class="mt-1 text-sm text-muted-foreground">Detalle del ítem de catálogo</p>
          </div>
          <a hlmBtn variant="outline" routerLink="edit">Editar</a>
        </header>

        <hlm-card class="space-y-4 p-6">
          <div class="grid gap-3 text-sm">
            <div>
              <p class="text-muted-foreground">Tipo</p>
              <span hlmBadge variant="secondary">{{ item.kind }}</span>
            </div>
            <div>
              <p class="text-muted-foreground">Estado</p>
              <p class="font-medium">{{ item.status }}</p>
            </div>
            <div>
              <p class="text-muted-foreground">SKU interno</p>
              <p class="font-medium">{{ item.internal_sku || '—' }}</p>
            </div>
            <div>
              <p class="text-muted-foreground">Creado</p>
              <p>{{ item.created_at | date: 'medium' }}</p>
            </div>
            <div>
              <p class="text-muted-foreground">Actualizado</p>
              <p>{{ item.updated_at | date: 'medium' }}</p>
            </div>
          </div>
        </hlm-card>
      }
    </div>
  `,
})
export class DetailItemPage implements OnInit {
  readonly store = inject(CatalogStore);
  private readonly route = inject(ActivatedRoute);

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('itemId');
    if (id) this.store.loadItem(id);
  }
}
