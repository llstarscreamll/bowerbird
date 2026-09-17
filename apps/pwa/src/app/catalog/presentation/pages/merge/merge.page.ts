import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { forkJoin } from 'rxjs';
import { NgIcon } from '@ng-icons/core';
import { BrnAlertDialogContent } from '@spartan-ng/brain/alert-dialog';
import { HlmAlertDialogImports } from '@spartan-ng/helm/alert-dialog';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { HlmTooltipImports } from '@spartan-ng/helm/tooltip';
import { CatalogStore } from '../../../application/catalog.store';
import { CatalogHttpService } from '../../../infrastructure/catalog.http.service';
import {
  CATALOG_MERGE_MAX_ITEMS,
  CATALOG_MERGE_MIN_ITEMS,
  CatalogAlias,
  CatalogItem,
  creationSourceLabel,
  kindLabel,
  isMergeInternalCodeInherited,
  pickDefaultSurvivor,
  previewMergeInternalCode,
  statusLabel,
  survivorSuggestionTooltip,
} from '../../../domain/catalog.model';

type AliasPreview = CatalogAlias & { keep: boolean; fromItemId: string };

@Component({
  selector: 'app-catalog-merge',
  standalone: true,
  imports: [FormsModule, RouterLink, NgIcon, BrnAlertDialogContent, HlmAlertDialogImports, HlmAlertImports, HlmBadgeImports, HlmButtonImports, HlmCardImports, HlmSpinnerImports, HlmTooltipImports],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-6xl space-y-6">
      <a routerLink=".." class="inline-flex items-center text-sm text-muted-foreground hover:text-foreground">
        <ng-icon name="lucideArrowLeft" class="mr-1" />
        Volver al catálogo
      </a>
      <header>
        <h1 class="text-2xl font-semibold tracking-tight">Fusionar ítems</h1>
        <p class="mt-1 text-sm text-muted-foreground">Elige cuál registro permanece. Los demás se absorben y sus códigos de cruce pasan al conservado.</p>
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

        <section class="space-y-3">
          <div>
            <h2 class="text-base font-semibold">Ítem que permanece</h2>
            <p class="text-sm text-muted-foreground">Haz clic en el registro que quedará en el catálogo.</p>
          </div>
          <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            @for (item of items(); track item.id) {
              <button
                type="button"
                class="rounded-lg border p-4 text-left transition-colors hover:bg-muted/40"
                [class.border-primary]="item.id === survivorId()"
                [class.ring-2]="item.id === survivorId()"
                [class.ring-primary]="item.id === survivorId()"
                [class.opacity-70]="item.id !== survivorId()"
                (click)="selectSurvivor(item.id)"
              >
                <div class="flex flex-wrap items-center gap-2">
                  @if (item.id === survivorId()) {
                    <span hlmBadge>Permanece</span>
                  } @else {
                    <span hlmBadge variant="secondary">Se fusionará</span>
                  }
                  @if (item.id === suggestedId()) {
                    <span hlmBadge variant="outline" class="cursor-help" [hlmTooltip]="suggestionTooltip(item)" tabindex="0" (click)="$event.stopPropagation()"> Sugerido </span>
                  }
                </div>
                <p class="mt-2 font-medium">{{ item.name }}</p>
                <p class="mt-1 font-mono text-xs text-muted-foreground">{{ item.internal_code || 'Sin código interno' }}</p>
                <p class="mt-2 text-xs text-muted-foreground">{{ statusLabel(item.status) }} · {{ creationSourceLabel(item.creation_source) }}</p>
              </button>
            }
          </div>
          @if (hasDistinctCodes() && survivor()?.internal_code) {
            <p class="text-sm text-muted-foreground">
              El código interno será <span class="font-mono">{{ survivor()?.internal_code }}</span> (del ítem que permanece). Para usar otro código, elige ese ítem arriba.
            </p>
          }
        </section>

        @if (hasAnyConflict()) {
          <hlm-card class="space-y-4 p-6">
            <div>
              <h2 class="text-base font-semibold">Resolver diferencias</h2>
              <p class="text-sm text-muted-foreground">Estos campos no coinciden entre los ítems. Elige el valor del resultado.</p>
            </div>

            @if (hasNameConflict()) {
              <div class="space-y-2">
                <p class="text-sm font-medium text-muted-foreground">Nombre</p>
                <div class="flex flex-col gap-2">
                  @for (item of items(); track item.id) {
                    <button
                      type="button"
                      class="rounded-md border px-3 py-2 text-left text-sm transition-colors hover:bg-muted/40"
                      [class.border-primary]="nameItemId() === item.id"
                      [class.ring-1]="nameItemId() === item.id"
                      [class.ring-primary]="nameItemId() === item.id"
                      (click)="nameItemId.set(item.id)"
                    >
                      {{ item.name }}
                    </button>
                  }
                </div>
              </div>
            }

            @if (hasKindConflict()) {
              <div class="space-y-2">
                <p class="text-sm font-medium text-muted-foreground">Tipo</p>
                <div class="flex flex-wrap gap-2">
                  @for (item of items(); track item.id) {
                    <button
                      type="button"
                      class="rounded-md border px-3 py-2 text-sm transition-colors hover:bg-muted/40"
                      [class.border-primary]="kindItemId() === item.id"
                      [class.ring-1]="kindItemId() === item.id"
                      [class.ring-primary]="kindItemId() === item.id"
                      (click)="kindItemId.set(item.id)"
                    >
                      {{ kindLabel(item.kind) }}
                    </button>
                  }
                </div>
              </div>
            }

            @if (needsCodePicker()) {
              <div class="space-y-2">
                <p class="text-sm font-medium text-muted-foreground">Código interno</p>
                <p class="text-xs text-muted-foreground">El ítem que permanece no tiene código. Elige cuál conservar.</p>
                <div class="flex flex-wrap gap-2">
                  @for (item of items(); track item.id) {
                    @if (item.internal_code) {
                      <button
                        type="button"
                        class="rounded-md border px-3 py-2 font-mono text-sm transition-colors hover:bg-muted/40"
                        [class.border-primary]="codeItemId() === item.id"
                        [class.ring-1]="codeItemId() === item.id"
                        [class.ring-primary]="codeItemId() === item.id"
                        (click)="codeItemId.set(item.id)"
                      >
                        {{ item.internal_code }}
                      </button>
                    }
                  }
                </div>
              </div>
            }
          </hlm-card>
        }

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

        @if (mergedPreview(); as preview) {
          <hlm-card class="space-y-4 p-6">
            <div>
              <h2 class="text-base font-semibold">Así quedará el ítem</h2>
              <p class="text-sm text-muted-foreground">Revisa el resultado antes de fusionar.</p>
            </div>
            <dl class="grid gap-2 text-sm sm:grid-cols-2">
              <div>
                <dt class="text-muted-foreground">Nombre</dt>
                <dd class="font-medium">{{ preview.name }}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground">Tipo</dt>
                <dd>{{ kindLabel(preview.kind) }}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground">Código interno</dt>
                <dd class="font-mono">
                  {{ preview.internal_code || '—' }}
                  @if (preview.internal_code_inherited) {
                    <span class="font-sans text-muted-foreground">· heredado al fusionar</span>
                  }
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground">Estado</dt>
                <dd>{{ statusLabel(preview.status) }}</dd>
              </div>
            </dl>
            @if (sourceItems().length > 0) {
              <div>
                <p class="text-sm text-muted-foreground">Desaparecerán del catálogo:</p>
                <ul class="mt-1 space-y-1 text-sm">
                  @for (item of sourceItems(); track item.id) {
                    <li class="font-mono text-muted-foreground">{{ item.internal_code || item.name }}</li>
                  }
                </ul>
              </div>
            }
            <p class="text-sm text-muted-foreground">{{ keptAliasCount() }} {{ keptAliasCount() === 1 ? 'código de cruce' : 'códigos de cruce' }} se conservarán.</p>
            <div hlmAlert>
              <ng-icon name="lucideInfo" hlmAlertIcon />
              <h4 hlmAlertTitle>Implicación en facturas</h4>
              <p hlmAlertDescription>
                Si estos ítems tienen líneas de factura vinculadas, quedarán asociadas al ítem conservado. El contenido de las facturas no cambia; solo se actualiza el vínculo al catálogo.
              </p>
            </div>
            <div class="flex flex-wrap gap-2">
              <button hlmBtn type="button" [disabled]="store.submitting() || !canMerge()" (click)="confirmOpen.set(true)">Fusionar</button>
              <a hlmBtn variant="outline" routerLink="..">Cancelar</a>
            </div>
          </hlm-card>
        }
      }
    </div>

    <hlm-alert-dialog [state]="confirmOpen() ? 'open' : 'closed'" (closed)="confirmOpen.set(false)">
      <hlm-alert-dialog-content *brnAlertDialogContent>
        <hlm-alert-dialog-header>
          <h2 hlmAlertDialogTitle>¿Fusionar {{ items().length }} ítems en uno?</h2>
          <p hlmAlertDialogDescription>
            @if (mergedPreview(); as preview) {
              Quedará <strong>{{ preview.name }}</strong>
              @if (preview.internal_code) {
                (<span class="font-mono">{{ preview.internal_code }}</span
                >)
              }
              .
            }
            <strong>Facturas:</strong> si alguno tiene líneas vinculadas, pasarán al ítem conservado (el contenido de las facturas no cambia).
            @if (sourceItems().length > 0) {
              {{ sourceItems().length }} {{ sourceItems().length === 1 ? 'registro desaparecerá' : 'registros desaparecerán' }} del catálogo.
            }
            Esta acción no se puede deshacer.
          </p>
        </hlm-alert-dialog-header>
        <hlm-alert-dialog-footer>
          <button hlmBtn variant="outline" type="button" (click)="confirmOpen.set(false)">Cancelar</button>
          <button hlmBtn type="button" [disabled]="store.submitting()" (click)="submit()">Fusionar</button>
        </hlm-alert-dialog-footer>
      </hlm-alert-dialog-content>
    </hlm-alert-dialog>
  `,
})
export class MergePage implements OnInit {
  readonly store = inject(CatalogStore);
  readonly creationSourceLabel = creationSourceLabel;
  readonly kindLabel = kindLabel;
  readonly statusLabel = statusLabel;
  private readonly http = inject(CatalogHttpService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  readonly items = signal<CatalogItem[]>([]);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly confirmOpen = signal(false);
  readonly suggestedId = signal('');
  readonly survivorId = signal('');
  readonly nameItemId = signal('');
  readonly kindItemId = signal('');
  readonly codeItemId = signal('');

  readonly survivor = computed(() => this.items().find((item) => item.id === this.survivorId()));

  readonly sourceItems = computed(() => this.items().filter((item) => item.id !== this.survivorId()));

  readonly hasNameConflict = computed(() => new Set(this.items().map((item) => item.name)).size > 1);

  readonly hasKindConflict = computed(() => new Set(this.items().map((item) => item.kind)).size > 1);

  readonly hasDistinctCodes = computed(() => {
    const codes = this.items()
      .map((item) => item.internal_code)
      .filter((code): code is string => Boolean(code));
    return new Set(codes).size > 1;
  });

  readonly needsCodePicker = computed(() => {
    const survivor = this.survivor();
    if (survivor?.internal_code) return false;
    return this.hasDistinctCodes();
  });

  readonly hasAnyConflict = computed(() => this.hasNameConflict() || this.hasKindConflict() || this.needsCodePicker());

  readonly mergedPreview = computed(() => {
    const survivor = this.survivor();
    if (!survivor) return null;
    const nameItem = this.items().find((item) => item.id === this.nameItemId());
    const kindItem = this.items().find((item) => item.id === this.kindItemId());
    const internalCode = previewMergeInternalCode(survivor, this.items(), this.codeItemId());
    return {
      name: nameItem?.name ?? survivor.name,
      kind: kindItem?.kind ?? survivor.kind,
      internal_code: internalCode,
      internal_code_inherited: isMergeInternalCodeInherited(survivor, this.items(), internalCode),
      status: survivor.status,
    };
  });

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

  ngOnInit(): void {
    const raw = this.route.snapshot.queryParamMap.get('ids') || '';
    const ids = [
      ...new Set(
        raw
          .split(',')
          .map((id) => id.trim())
          .filter(Boolean),
      ),
    ];
    if (ids.length < CATALOG_MERGE_MIN_ITEMS || ids.length > CATALOG_MERGE_MAX_ITEMS) {
      this.loading.set(false);
      this.error.set(`Selecciona entre ${CATALOG_MERGE_MIN_ITEMS} y ${CATALOG_MERGE_MAX_ITEMS} ítems para fusionar.`);
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

  suggestionTooltip(item: CatalogItem): string {
    return survivorSuggestionTooltip(item, this.items());
  }

  selectSurvivor(id: string): void {
    this.survivorId.set(id);
    this.error.set(null);
    const item = this.items().find((row) => row.id === id);
    if (item?.internal_code) this.codeItemId.set(id);
  }

  canMerge(): boolean {
    if (!this.survivorId()) return false;
    if (this.needsCodePicker() && !this.codeItemId()) return false;
    return true;
  }

  submit(): void {
    const survivor = this.survivor();
    const nameItem = this.items().find((item) => item.id === this.nameItemId());
    const kindItem = this.items().find((item) => item.id === this.kindItemId());
    const codeItem = this.items().find((item) => item.id === this.codeItemId());
    if (!survivor) return;
    if (this.needsCodePicker() && codeItem?.internal_code && survivor.internal_code && codeItem.internal_code !== survivor.internal_code) {
      this.error.set('El código interno del ítem conservado no se puede cambiar. Conserva el ítem que tiene el código que quieres.');
      return;
    }
    const chosenCode = previewMergeInternalCode(survivor, this.items(), this.codeItemId()) || undefined;
    this.confirmOpen.set(false);
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
