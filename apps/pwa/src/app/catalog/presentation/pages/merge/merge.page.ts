import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { forkJoin } from 'rxjs';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { CatalogStore } from '../../../application/catalog.store';
import { CatalogHttpService } from '../../../infrastructure/catalog.http.service';
import { CatalogAlias, CatalogItem, CATALOG_KINDS, creationSourceLabel, pickDefaultSurvivor } from '../../../domain/catalog.model';

type AliasPreview = CatalogAlias & { keep: boolean; fromItemId: string };

@Component({
  selector: 'app-catalog-merge',
  standalone: true,
  imports: [FormsModule, RouterLink, NgIcon, HlmAlertImports, HlmBadgeImports, HlmButtonImports, HlmCardImports, HlmSpinnerImports],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-6xl space-y-6">
      <a routerLink=".." class="inline-flex items-center text-sm text-muted-foreground hover:text-foreground">
        <ng-icon name="lucideArrowLeft" class="mr-1" />
        Volver al catálogo
      </a>
      <header>
        <h1 class="text-2xl font-semibold tracking-tight">Fusionar ítems</h1>
        <p class="mt-1 text-sm text-muted-foreground">Elige el ítem que se conserva. El cruce de códigos se une automáticamente.</p>
      </header>

      @if (error(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }
      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      @if (loading()) {
        <div class="flex justify-center py-16"><hlm-spinner class="size-8" /></div>
      } @else if (items().length >= 2) {
        @if (variantWarning()) {
          <div hlmAlert>
            <ng-icon name="lucideTriangleAlert" hlmAlertIcon />
            <h4 hlmAlertTitle>Pueden ser presentación o empaque distinto</h4>
            <p hlmAlertDescription>Un ítem puede quedar con dos GTIN. Fusiona solo si son el mismo producto.</p>
          </div>
        }

        <div class="overflow-x-auto">
          <table class="w-full min-w-[40rem] text-sm">
            <thead>
              <tr class="border-b border-border text-left">
                <th class="py-2 pr-3 font-medium text-muted-foreground">Campo</th>
                @for (item of items(); track item.id) {
                  <th class="py-2 pr-3 font-medium">
                    <label class="flex cursor-pointer items-center gap-2">
                      <input type="radio" name="survivor" [value]="item.id" [ngModel]="survivorId()" (ngModelChange)="survivorId.set($event)" />
                      Se conserva
                    </label>
                    @if (item.id === suggestedId()) {
                      <span hlmBadge class="mt-1" variant="secondary">Sugerido</span>
                    }
                  </th>
                }
              </tr>
            </thead>
            <tbody>
              <tr class="border-b border-border">
                <td class="py-2 pr-3 text-muted-foreground">Nombre</td>
                @for (item of items(); track item.id) {
                  <td class="py-2 pr-3">
                    <label class="flex cursor-pointer items-start gap-2">
                      <input type="radio" name="name" [value]="item.id" [ngModel]="nameItemId()" (ngModelChange)="nameItemId.set($event)" />
                      <span>{{ item.name }}</span>
                    </label>
                  </td>
                }
              </tr>
              <tr class="border-b border-border">
                <td class="py-2 pr-3 text-muted-foreground">Tipo</td>
                @for (item of items(); track item.id) {
                  <td class="py-2 pr-3">
                    <label class="flex cursor-pointer items-center gap-2">
                      <input type="radio" name="kind" [value]="item.id" [ngModel]="kindItemId()" (ngModelChange)="kindItemId.set($event)" />
                      {{ kindLabel(item.kind) }}
                    </label>
                  </td>
                }
              </tr>
              <tr class="border-b border-border">
                <td class="py-2 pr-3 text-muted-foreground">Código interno</td>
                @for (item of items(); track item.id) {
                  <td class="py-2 pr-3">
                    @if (needsCodeChoice()) {
                      <label class="flex cursor-pointer items-center gap-2">
                        <input type="radio" name="code" [value]="item.id" [ngModel]="codeItemId()" (ngModelChange)="codeItemId.set($event)" [disabled]="!item.internal_code" />
                        {{ item.internal_code || '—' }}
                      </label>
                    } @else {
                      {{ item.internal_code || '—' }}
                    }
                  </td>
                }
              </tr>
              <tr class="border-b border-border">
                <td class="py-2 pr-3 text-muted-foreground">Estado</td>
                @for (item of items(); track item.id) {
                  <td class="py-2 pr-3">{{ item.status }}</td>
                }
              </tr>
              <tr>
                <td class="py-2 pr-3 text-muted-foreground">Origen</td>
                @for (item of items(); track item.id) {
                  <td class="py-2 pr-3">{{ creationSourceLabel(item.creation_source) }}</td>
                }
              </tr>
            </tbody>
          </table>
        </div>

        <hlm-card class="space-y-3 p-6">
          <div>
            <h2 class="text-base font-semibold">Cruce de códigos</h2>
            <p class="text-sm text-muted-foreground">SKU de proveedor y GTIN se unen en el ítem conservado. No se eligen uno a uno.</p>
          </div>
          @if (aliasPreview().length === 0) {
            <p class="text-sm text-muted-foreground">Sin códigos de cruce.</p>
          } @else {
            <div class="overflow-x-auto">
              <table class="w-full text-left text-sm">
                <thead class="text-xs uppercase text-muted-foreground">
                  <tr>
                    <th class="py-2 pr-3 font-medium">Esquema</th>
                    <th class="py-2 pr-3 font-medium">Valor</th>
                    <th class="py-2 pr-3 font-medium">Contacto</th>
                    <th class="py-2 pr-3 font-medium">Origen</th>
                    <th class="py-2 font-medium">Resultado</th>
                  </tr>
                </thead>
                <tbody>
                  @for (alias of aliasPreview(); track alias.id) {
                    <tr class="border-t border-border">
                      <td class="py-2 pr-3 font-mono">{{ alias.scheme }}</td>
                      <td class="py-2 pr-3 font-mono">{{ alias.value }}</td>
                      <td class="py-2 pr-3 font-mono">{{ alias.party_id || '—' }}</td>
                      <td class="py-2 pr-3">{{ alias.source === 'manual' ? 'Manual' : 'Factura' }}</td>
                      <td class="py-2">{{ alias.keep ? 'Se conserva' : 'Ya existe en el conservado' }}</td>
                    </tr>
                  }
                </tbody>
              </table>
            </div>
          }
        </hlm-card>

        <p class="text-sm text-muted-foreground">
          @if (lineCount() > 0) {
            {{ lineCount() }} líneas de factura se unificarán ·
          }
          {{ keptAliasCount() }} códigos de cruce se conservarán.
        </p>

        <div class="flex flex-wrap gap-2">
          <button hlmBtn type="button" [disabled]="store.submitting() || !canMerge()" (click)="submit()">Fusionar</button>
          <a hlmBtn variant="outline" routerLink="..">Cancelar</a>
        </div>
      }
    </div>
  `,
})
export class MergePage implements OnInit {
  readonly store = inject(CatalogStore);
  readonly creationSourceLabel = creationSourceLabel;
  private readonly http = inject(CatalogHttpService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  readonly items = signal<CatalogItem[]>([]);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly suggestedId = signal('');
  readonly survivorId = signal('');
  readonly nameItemId = signal('');
  readonly kindItemId = signal('');
  readonly codeItemId = signal('');

  readonly variantWarning = computed(() => {
    const gtins = new Set<string>();
    for (const item of this.items()) {
      for (const alias of item.aliases || []) {
        if (alias.scheme === 'gtin') gtins.add(alias.value);
      }
    }
    const itemsWithGtin = this.items().filter((item) => (item.aliases || []).some((alias) => alias.scheme === 'gtin'));
    return itemsWithGtin.length >= 2 && gtins.size >= 2;
  });

  readonly aliasPreview = computed((): AliasPreview[] => {
    const survivor = this.survivorId();
    const have = new Set<string>();
    const out: AliasPreview[] = [];
    const ordered = [...this.items()].sort((a, b) => Number(b.id === survivor) - Number(a.id === survivor));
    for (const item of ordered) {
      for (const alias of item.aliases || []) {
        const key = `${alias.scheme}|${alias.party_id || ''}|${alias.value}`;
        const keep = !have.has(key);
        have.add(key);
        out.push({ ...alias, keep, fromItemId: item.id });
      }
    }
    return out;
  });

  readonly keptAliasCount = computed(() => this.aliasPreview().filter((alias) => alias.keep).length);

  readonly lineCount = computed(() => {
    const ids = new Set(this.items().map((item) => item.id));
    let n = 0;
    for (const cluster of this.store.duplicateClusters()) {
      for (const member of cluster.items) {
        if (ids.has(member.id)) n += member.line_count || 0;
      }
    }
    return n;
  });

  readonly needsCodeChoice = computed(() => {
    const codes = this.items()
      .map((item) => item.internal_code)
      .filter((code): code is string => Boolean(code));
    return new Set(codes).size > 1;
  });

  ngOnInit(): void {
    this.store.loadDuplicateClusters();
    const raw = this.route.snapshot.queryParamMap.get('ids') || '';
    const ids = [
      ...new Set(
        raw
          .split(',')
          .map((id) => id.trim())
          .filter(Boolean),
      ),
    ];
    if (ids.length < 2 || ids.length > 5) {
      this.loading.set(false);
      this.error.set('Selecciona entre 2 y 5 ítems para fusionar.');
      return;
    }
    forkJoin(ids.map((id) => this.http.getItem(id))).subscribe({
      next: (items) => this.setItems(items),
      error: () => {
        this.loading.set(false);
        this.error.set('No se pudieron cargar los ítems.');
      },
    });
  }

  canMerge(): boolean {
    if (!this.survivorId()) return false;
    if (this.needsCodeChoice() && !this.codeItemId()) return false;
    return true;
  }

  kindLabel(kind: string): string {
    return CATALOG_KINDS.find((row) => row.value === kind)?.label || kind;
  }

  submit(): void {
    const survivor = this.items().find((item) => item.id === this.survivorId());
    const nameItem = this.items().find((item) => item.id === this.nameItemId());
    const kindItem = this.items().find((item) => item.id === this.kindItemId());
    const codeItem = this.items().find((item) => item.id === this.codeItemId());
    if (!survivor) return;
    if (this.needsCodeChoice() && codeItem?.internal_code && survivor.internal_code && codeItem.internal_code !== survivor.internal_code) {
      this.error.set('El código interno del ítem conservado no se puede cambiar. Conserva el ítem que tiene el código que quieres.');
      return;
    }
    const chosenCode = this.needsCodeChoice() ? codeItem?.internal_code || undefined : survivor.internal_code || undefined;
    this.store
      .mergeItems({
        survivor_id: survivor.id,
        source_ids: this.items()
          .filter((item) => item.id !== survivor.id)
          .map((item) => item.id),
        name: nameItem?.name,
        kind: kindItem?.kind,
        internal_code: chosenCode || undefined,
      })
      .subscribe((item) => {
        if (item) void this.router.navigate(['../', item.id], { relativeTo: this.route });
      });
  }

  private setItems(items: CatalogItem[]): void {
    this.items.set(items);
    const suggested = pickDefaultSurvivor(items);
    this.suggestedId.set(suggested);
    this.survivorId.set(suggested);
    this.nameItemId.set(defaultNameId(items, suggested));
    this.kindItemId.set(items.find((item) => item.kind !== 'unknown')?.id || suggested);
    this.codeItemId.set(items.find((item) => item.id === suggested && item.internal_code)?.id || items.find((item) => item.internal_code)?.id || '');
    this.loading.set(false);
  }
}

function defaultNameId(items: CatalogItem[], survivorId: string): string {
  const survivor = items.find((item) => item.id === survivorId);
  if (!survivor) return items[0]?.id || '';
  if (survivor.creation_source === 'invoice') {
    const longest = [...items].sort((a, b) => b.name.length - a.name.length)[0];
    if (longest && longest.name.length > survivor.name.length) return longest.id;
  }
  return survivor.id;
}
