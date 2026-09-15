import { DatePipe } from '@angular/common';
import { Component, OnInit, effect, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { forkJoin, of } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { generateUlid } from '../../../../core/utils/ulid';
import { PartiesHttpService } from '../../../../parties/infrastructure/parties.http.service';
import { CatalogStore } from '../../../application/catalog.store';
import { CatalogAlias, creationSourceLabel } from '../../../domain/catalog.model';

@Component({
  selector: 'app-catalog-item-detail',
  standalone: true,
  imports: [DatePipe, FormsModule, RouterLink, NgIcon, HlmCardImports, HlmButtonImports, HlmBadgeImports, HlmSpinnerImports, HlmAlertImports],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-2xl space-y-6">
      <a routerLink=".." class="inline-flex items-center text-sm text-muted-foreground hover:text-foreground">
        <ng-icon name="lucideArrowLeft" class="mr-1" />
        Volver al catálogo
      </a>

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
          @if (store.conflictOwnerItemId(); as ownerId) {
            <p class="mt-2 text-sm">
              El alias ya existe en
              <a class="font-medium underline" [routerLink]="['/', tenantId(), 'catalog', ownerId]">otro ítem</a>.
            </p>
            <a class="mt-3 inline-flex" hlmBtn size="sm" [routerLink]="['/', tenantId(), 'catalog', 'merge']" [queryParams]="{ ids: mergeIds(ownerId) }"> Fusionar con este ítem </a>
          }
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
              <p class="text-muted-foreground">Origen</p>
              <p class="font-medium">{{ creationSourceLabel(item.creation_source) }}</p>
            </div>
            <div>
              <p class="text-muted-foreground">Estado</p>
              <p class="font-medium">{{ item.status }}</p>
            </div>
            <div>
              <p class="text-muted-foreground">Código interno</p>
              <p class="font-medium">{{ item.internal_code || '—' }}</p>
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

        <hlm-card class="space-y-4 p-6">
          <div>
            <h2 class="text-base font-semibold">Cruce de códigos</h2>
            <p class="text-sm text-muted-foreground">SKU de proveedor y GTIN que identifican este ítem en facturas.</p>
          </div>

          @if ((item.aliases || []).length === 0) {
            <p class="text-sm text-muted-foreground">Sin aliases todavía.</p>
          } @else {
            <div class="overflow-x-auto">
              <table class="w-full text-left text-sm">
                <thead class="text-xs uppercase text-muted-foreground">
                  <tr>
                    <th class="py-2 pr-3 font-medium">Esquema</th>
                    <th class="py-2 pr-3 font-medium">Valor</th>
                    <th class="py-2 pr-3 font-medium">Contacto</th>
                    <th class="py-2 pr-3 font-medium">Origen</th>
                    <th class="py-2 font-medium"></th>
                  </tr>
                </thead>
                <tbody>
                  @for (alias of item.aliases || []; track alias.id) {
                    <tr class="border-t border-border">
                      <td class="py-2 pr-3 font-mono">{{ alias.scheme }}</td>
                      <td class="py-2 pr-3 font-mono">{{ alias.value }}</td>
                      <td class="py-2 pr-3">{{ partyLabel(alias) }}</td>
                      <td class="py-2 pr-3">{{ alias.source === 'manual' ? 'Manual' : 'Factura' }}</td>
                      <td class="py-2 text-right">
                        <button hlmBtn size="sm" variant="ghost" [disabled]="store.submitting()" (click)="removeAlias(alias)">Quitar</button>
                      </td>
                    </tr>
                  }
                </tbody>
              </table>
            </div>
          }

          <form class="grid gap-3 sm:grid-cols-2" (ngSubmit)="addAlias()">
            <div class="space-y-1.5">
              <label class="text-sm font-medium" for="alias-scheme">Esquema</label>
              <select id="alias-scheme" name="scheme" class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm" [(ngModel)]="scheme">
                <option value="supplier_sku">SKU proveedor</option>
                <option value="gtin">GTIN</option>
              </select>
            </div>
            <div class="space-y-1.5">
              <label class="text-sm font-medium" for="alias-value">Valor</label>
              <input id="alias-value" name="value" [(ngModel)]="value" required class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm" />
            </div>
            @if (scheme === 'supplier_sku') {
              <div class="space-y-1.5 sm:col-span-2">
                <label class="text-sm font-medium" for="alias-party">ID de contacto</label>
                <input id="alias-party" name="party_id" [(ngModel)]="partyId" required class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono" />
              </div>
            }
            <div class="sm:col-span-2">
              <button hlmBtn type="submit" [disabled]="store.submitting()">Añadir alias</button>
            </div>
          </form>
        </hlm-card>
      }
    </div>
  `,
})
export class DetailItemPage implements OnInit {
  readonly store = inject(CatalogStore);
  readonly creationSourceLabel = creationSourceLabel;
  private readonly route = inject(ActivatedRoute);
  private readonly parties = inject(PartiesHttpService);

  readonly partyNames = signal<Record<string, string>>({});
  scheme = 'supplier_sku';
  value = '';
  partyId = '';

  constructor() {
    effect(() => {
      const aliases = this.store.selectedItem()?.aliases || [];
      this.resolveParties(aliases);
    });
  }

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('itemId');
    if (id) this.store.loadItem(id);
  }

  tenantId(): string {
    return this.route.parent?.snapshot.paramMap.get('tenantId') || this.route.snapshot.paramMap.get('tenantId') || location.pathname.split('/').filter(Boolean)[0] || '';
  }

  mergeIds(ownerId: string): string {
    const current = this.store.selectedItem()?.id;
    return [current, ownerId].filter(Boolean).join(',');
  }

  partyLabel(alias: CatalogAlias): string {
    const id = alias.party_id?.trim();
    if (!id) return '—';
    return this.partyNames()[id] || id;
  }

  addAlias(): void {
    const item = this.store.selectedItem();
    if (!item || !this.value.trim()) return;
    this.store
      .addAlias(item.id, {
        id: generateUlid(),
        scheme: this.scheme,
        value: this.value.trim(),
        party_id: this.scheme === 'supplier_sku' ? this.partyId.trim() : undefined,
      })
      .subscribe((ok) => {
        if (ok) {
          this.value = '';
          this.resolveParties(this.store.selectedItem()?.aliases || []);
        }
      });
  }

  removeAlias(alias: CatalogAlias): void {
    const item = this.store.selectedItem();
    if (!item) return;
    this.store.removeAlias(item.id, alias.id).subscribe();
  }

  private resolveParties(aliases: CatalogAlias[]): void {
    const ids = [...new Set(aliases.map((a) => a.party_id?.trim()).filter((id): id is string => Boolean(id)))];
    if (!ids.length) return;
    forkJoin(ids.map((id) => this.parties.getParty(id).pipe(catchError(() => of(null))))).subscribe((parties) => {
      const names: Record<string, string> = {};
      parties.forEach((party, i) => {
        if (party) names[ids[i]] = party.name;
      });
      this.partyNames.update((current) => ({ ...current, ...names }));
    });
  }
}
