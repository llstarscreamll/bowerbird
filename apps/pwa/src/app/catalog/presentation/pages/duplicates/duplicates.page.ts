import { Component, OnInit, inject } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { CatalogStore } from '../../../application/catalog.store';
import { DuplicateCluster, clusterReasonLabel, creationSourceLabel } from '../../../domain/catalog.model';

@Component({
  selector: 'app-catalog-duplicates',
  standalone: true,
  imports: [RouterLink, NgIcon, HlmAlertImports, HlmBadgeImports, HlmButtonImports, HlmCardImports, HlmSpinnerImports],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-3xl space-y-6">
      <a routerLink=".." class="inline-flex items-center text-sm text-muted-foreground hover:text-foreground">
        <ng-icon name="lucideArrowLeft" class="mr-1" />
        Volver al catálogo
      </a>
      <header>
        <h1 class="text-2xl font-semibold tracking-tight">Resolver duplicados</h1>
        <p class="mt-1 text-sm text-muted-foreground">Grupos que parecen el mismo producto. Revisa antes de fusionar.</p>
      </header>

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      @if (store.loading() && store.duplicateClusters().length === 0) {
        <div class="flex justify-center py-16"><hlm-spinner class="size-8" /></div>
      } @else if (store.duplicateClusters().length === 0) {
        <hlm-card class="p-8 text-center">
          <p class="text-sm text-muted-foreground">No hay duplicados pendientes.</p>
          <p class="mt-2 text-sm text-muted-foreground">Puedes fusionar ítems desde el listado con los recuadros de selección.</p>
          <a hlmBtn class="mt-4" routerLink="..">Ir al catálogo</a>
        </hlm-card>
      } @else {
        <div class="space-y-4">
          @for (cluster of store.duplicateClusters(); track cluster.id) {
            <hlm-card class="space-y-4 p-6">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <span hlmBadge variant="secondary">{{ clusterReasonLabel(cluster.reason) }}</span>
                <div class="flex flex-wrap gap-2">
                  <button hlmBtn size="sm" type="button" (click)="review(cluster)">Revisar</button>
                  <button hlmBtn size="sm" variant="outline" type="button" [disabled]="store.submitting()" (click)="dismiss(cluster)">No es el mismo producto</button>
                </div>
              </div>
              <ul class="space-y-3">
                @for (item of cluster.items; track item.id) {
                  <li class="rounded-md border border-border p-3 text-sm">
                    <p class="font-medium">{{ item.name }}</p>
                    <p class="mt-1 text-muted-foreground">
                      {{ item.internal_code || 'Sin código interno' }} · {{ creationSourceLabel(item.creation_source) }} · {{ item.status }} · {{ item.line_count }} líneas
                    </p>
                    @if (item.aliases?.length) {
                      <p class="mt-1 font-mono text-xs text-muted-foreground">
                        @for (alias of item.aliases; track alias.id; let last = $last) {
                          {{ alias.scheme }} {{ alias.value }}{{ last ? '' : ' · ' }}
                        }
                      </p>
                    }
                  </li>
                }
              </ul>
            </hlm-card>
          }
        </div>
      }
    </div>
  `,
})
export class DuplicatesPage implements OnInit {
  readonly store = inject(CatalogStore);
  readonly clusterReasonLabel = clusterReasonLabel;
  readonly creationSourceLabel = creationSourceLabel;
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  ngOnInit(): void {
    this.store.loadDuplicateClusters();
  }

  review(cluster: DuplicateCluster): void {
    void this.router.navigate(['../merge'], {
      relativeTo: this.route,
      queryParams: { ids: cluster.item_ids.join(',') },
    });
  }

  dismiss(cluster: DuplicateCluster): void {
    this.store.markNotDuplicates(cluster.item_ids).subscribe();
  }
}
