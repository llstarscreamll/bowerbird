import { CommonModule } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { CatalogStore } from '../../../application/catalog.store';
import { CatalogLinkerComponent } from '../../components/catalog-linker/catalog-linker.component';

@Component({
  selector: 'app-catalog-review',
  standalone: true,
  imports: [CommonModule, RouterLink, NgIcon, HlmCardImports, HlmSpinnerImports, HlmAlertImports, CatalogLinkerComponent],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-5xl space-y-6">
      <a routerLink="../" class="inline-flex items-center text-sm text-muted-foreground hover:text-foreground">
        <ng-icon name="lucideArrowLeft" class="mr-1" />
        Volver al catálogo
      </a>
      <header>
        <h1 class="text-2xl font-semibold tracking-tight">Cola de revisión</h1>
        <p class="mt-1 text-sm text-muted-foreground">Líneas sin vincular o con sugerencias. Confirma, recuerda y bloquea la decisión.</p>
      </header>

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      @if (store.loading()) {
        <div class="flex justify-center py-16"><hlm-spinner class="size-8" /></div>
      } @else {
        <div class="space-y-4">
          @for (line of store.reviewLines(); track line.id) {
            <hlm-card class="space-y-3 p-4">
              <div class="flex flex-wrap items-start justify-between gap-2">
                <div>
                  <p class="font-medium">{{ line.description || 'Sin descripción' }}</p>
                  <p class="text-xs text-muted-foreground">Código: {{ line.item_code || '—' }} · Estado: {{ line.link_status }}</p>
                </div>
                <a class="text-xs text-primary underline" [routerLink]="['/', tenantPrefix(), 'invoices', line.invoice_header_id]">Ver factura</a>
              </div>
              <app-catalog-linker [lineId]="line.id" [description]="line.description" [itemCode]="line.item_code" [suggestions]="line.suggestions || []" (resolved)="onResolved()" />
            </hlm-card>
          } @empty {
            <p class="py-10 text-center text-muted-foreground">No hay líneas pendientes de revisión.</p>
          }
        </div>
      }
    </div>
  `,
})
export class ReviewPage implements OnInit {
  readonly store = inject(CatalogStore);

  ngOnInit(): void {
    this.store.loadReviewQueue();
  }

  tenantPrefix(): string {
    return location.pathname.split('/')[1] || '';
  }

  onResolved(): void {
    this.store.loadReviewQueue();
  }
}
