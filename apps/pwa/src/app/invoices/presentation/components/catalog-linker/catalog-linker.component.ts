import { DecimalPipe } from '@angular/common';
import { Component, DestroyRef, OnInit, computed, inject, input, output, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { NgIcon } from '@ng-icons/core';
import { Subject, debounceTime, distinctUntilChanged, switchMap } from 'rxjs';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCheckboxImports } from '@spartan-ng/helm/checkbox';
import { HlmCommandImports } from '@spartan-ng/helm/command';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { InvoicesStore } from '../../../application/invoices.store';
import { CatalogSearchHit, LineDecisionPayload, rememberPreview } from '../../../domain/invoice.model';

@Component({
  selector: 'app-catalog-linker',
  standalone: true,
  imports: [DecimalPipe, FormsModule, NgIcon, HlmBadgeImports, HlmButtonImports, HlmCheckboxImports, HlmCommandImports, HlmSpinnerImports],
  host: { class: 'block space-y-3' },
  template: `
    @if (locked()) {
      <button hlmBtn size="sm" variant="outline" [disabled]="busy()" (click)="unlock()">Desbloquear</button>
    } @else {
      @if (suggestions().length) {
        <div class="space-y-2">
          <p class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Sugerencias</p>
          <ul class="space-y-1.5">
            @for (s of suggestions(); track s.item_id) {
              <li class="flex items-center justify-between gap-2 rounded-md border border-border bg-background px-2.5 py-2">
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium">{{ s.name || shortId(s.item_id) }}</p>
                  <p class="text-xs text-muted-foreground">{{ (s.score * 100 | number: '1.0-0') + '%' }} · {{ s.reason || 'soft' }}</p>
                </div>
                <button hlmBtn size="sm" variant="outline" [disabled]="busy()" (click)="link(s.item_id)">Vincular</button>
              </li>
            }
          </ul>
        </div>
      }

      <div class="space-y-2">
        <p class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Buscar en catálogo</p>
        <hlm-command class="min-h-40 rounded-lg border border-border" [filter]="passthroughFilter" (searchChange)="onSearch($event)">
          <hlm-command-input placeholder="Nombre, código interno, SKU o GTIN…" />
          <hlm-command-list>
            <div *hlmCommandEmptyState hlmCommandEmpty>
              @if (searching()) {
                <hlm-spinner class="size-4" />
              } @else {
                Sin resultados.
              }
            </div>
            <hlm-command-group>
              @for (item of searchResults(); track item.id) {
                <button hlm-command-item [value]="item.id + ' ' + item.name" (selected)="link(item.id)" [disabled]="busy()">
                  <span class="truncate">{{ item.name }}</span>
                  <span hlmBadge variant="outline" class="ms-auto shrink-0">{{ item.status }}</span>
                </button>
              }
            </hlm-command-group>
          </hlm-command-list>
        </hlm-command>
      </div>

      <div class="flex flex-wrap items-center gap-4 text-sm">
        <label class="inline-flex items-center gap-2">
          <hlm-checkbox [checked]="remember()" (checkedChange)="remember.set($event === true)" aria-label="Recordar" />
          Recordar
        </label>
        <label class="inline-flex items-center gap-2">
          <hlm-checkbox [checked]="lock()" (checkedChange)="lock.set($event === true)" aria-label="Bloquear esta línea" />
          Bloquear esta línea
        </label>
      </div>

      @if (remember() && previewText()) {
        <p class="text-xs text-muted-foreground">Se adjuntarán: {{ previewText() }}</p>
      }

      <div class="flex flex-wrap gap-2">
        <button hlmBtn size="sm" variant="outline" [disabled]="busy()" (click)="createProvisional()">
          <ng-icon name="lucideCirclePlus" />
          Nuevo provisional
        </button>
        <button hlmBtn size="sm" variant="destructive" [disabled]="busy()" (click)="reject()">
          <ng-icon name="lucideUnlink" />
          Rechazar
        </button>
      </div>
    }
  `,
})
export class CatalogLinkerComponent implements OnInit {
  private readonly invoicesStore = inject(InvoicesStore);
  private readonly destroyRef = inject(DestroyRef);
  private readonly search$ = new Subject<string>();

  readonly lineId = input.required<string>();
  readonly invoiceId = input.required<string>();
  readonly description = input('');
  readonly sellerSku = input('');
  readonly gtin = input('');
  readonly locked = input(false);
  readonly suggestions = input<{ item_id: string; name?: string; score: number; reason: string }[]>([]);
  readonly resolved = output<void>();

  readonly searchResults = signal<CatalogSearchHit[]>([]);
  readonly searching = signal(false);
  readonly busy = signal(false);
  readonly remember = signal(true);
  readonly lock = signal(true);

  readonly passthroughFilter = () => true;
  readonly previewText = computed(() => {
    const preview = rememberPreview(this.sellerSku(), this.gtin());
    const parts: string[] = [];
    if (preview.seller) parts.push(`SKU ${preview.seller}`);
    if (preview.gtin) parts.push(`GTIN ${preview.gtin}`);
    return parts.join(', ');
  });

  ngOnInit(): void {
    this.search$
      .pipe(
        debounceTime(250),
        distinctUntilChanged(),
        switchMap((query) => {
          this.searching.set(true);
          return this.invoicesStore.searchCatalogItems(query);
        }),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe((items) => {
        this.searchResults.set(items);
        this.searching.set(false);
      });
  }

  onSearch(query: string): void {
    this.search$.next(query?.trim() ?? '');
  }

  shortId(id: string): string {
    if (!id) return '—';
    return id.length > 12 ? `${id.slice(0, 8)}…${id.slice(-4)}` : id;
  }

  link(itemId: string): void {
    const id = itemId?.trim();
    if (!id || this.busy()) return;
    this.run({ item_id: id, action: 'link', remember: this.remember(), lock: this.lock() });
  }

  reject(): void {
    const blocked = this.suggestions()[0]?.item_id;
    this.run({ item_id: blocked, action: 'never_match', remember: this.remember(), lock: this.lock() });
  }

  createProvisional(): void {
    this.run({ action: 'create_provisional', remember: true, lock: true });
  }

  unlock(): void {
    this.run({ action: 'unlock', remember: false, lock: false });
  }

  private run(payload: LineDecisionPayload): void {
    this.busy.set(true);
    this.invoicesStore.resolveLineDecision(this.invoiceId(), this.lineId(), payload).subscribe((ok) => {
      this.busy.set(false);
      if (ok) this.resolved.emit();
    });
  }
}
