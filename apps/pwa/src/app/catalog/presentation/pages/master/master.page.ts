import { CommonModule } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { HlmTableImports } from '@spartan-ng/helm/table';
import { CatalogStore } from '../../../application/catalog.store';

@Component({
  selector: 'app-catalog-master',
  standalone: true,
  imports: [CommonModule, RouterLink, NgIcon, HlmCardImports, HlmSpinnerImports, HlmTableImports, HlmAlertImports, HlmBadgeImports, HlmButtonImports],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-5xl space-y-6">
      <header class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">Catálogo</h1>
          <p class="mt-1 text-sm text-muted-foreground">Ítems (productos, servicios, activos) vinculados desde líneas de factura.</p>
        </div>
        <a hlmBtn variant="outline" routerLink="review">Cola de revisión</a>
      </header>

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      <hlm-card class="overflow-hidden p-0">
        @if (store.loading()) {
          <div class="flex justify-center py-16"><hlm-spinner class="size-8" /></div>
        } @else {
          <table hlmTable>
            <thead hlmTHead>
              <tr hlmTr>
                <th hlmTh>Nombre</th>
                <th hlmTh>Tipo</th>
                <th hlmTh>Estado</th>
              </tr>
            </thead>
            <tbody hlmTBody>
              @for (item of store.items(); track item.id) {
                <tr hlmTr>
                  <td hlmTd class="font-medium">{{ item.name }}</td>
                  <td hlmTd>
                    <span hlmBadge variant="secondary">{{ item.kind }}</span>
                  </td>
                  <td hlmTd>{{ item.status }}</td>
                </tr>
              } @empty {
                <tr hlmTr>
                  <td hlmTd colspan="3" class="py-10 text-center text-muted-foreground">Aún no hay ítems en el catálogo.</td>
                </tr>
              }
            </tbody>
          </table>
        }
      </hlm-card>
    </div>
  `,
})
export class MasterPage implements OnInit {
  readonly store = inject(CatalogStore);

  ngOnInit(): void {
    this.store.loadItems();
  }
}
