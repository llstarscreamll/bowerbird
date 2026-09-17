import { DatePipe } from '@angular/common';
import { Component, DestroyRef, OnInit, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { interval } from 'rxjs';
import { BrnAlertDialogContent } from '@spartan-ng/brain/alert-dialog';
import { HlmAlertDialogImports } from '@spartan-ng/helm/alert-dialog';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmProgressImports } from '@spartan-ng/helm/progress';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { HlmTableImports } from '@spartan-ng/helm/table';
import { CatalogImportStore } from '../../../application/catalog-import.store';
import { catalogImportColumnLabel, catalogImportIsActive, catalogImportProgressPercent, catalogImportStatusLabel } from '../../../domain/catalog-import.model';

@Component({
  selector: 'app-catalog-import-detail',
  standalone: true,
  imports: [
    DatePipe,
    RouterLink,
    NgIcon,
    HlmCardImports,
    HlmSpinnerImports,
    HlmTableImports,
    HlmAlertImports,
    HlmBadgeImports,
    HlmButtonImports,
    HlmProgressImports,
    HlmAlertDialogImports,
    BrnAlertDialogContent,
  ],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-5xl space-y-6">
      <a hlmBtn variant="ghost" size="sm" routerLink=".." class="-ms-2">
        <ng-icon name="lucideArrowLeft" />
        Cargas masivas
      </a>

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      @if (store.loading() && !imp()) {
        <div class="flex justify-center py-16"><hlm-spinner class="size-8" /></div>
      } @else if (imp(); as item) {
        <header class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div class="min-w-0">
            <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">Carga masiva</h1>
            <p class="mt-1 min-w-0 text-sm text-muted-foreground">
              Solicitada por
              <span class="font-medium text-foreground">{{ item.requested_by.name }}</span>
              · {{ item.created_at | date: 'medium' }}
              <span class="block truncate">{{ item.requested_by.email }}</span>
            </p>
            @if (item.cancelled_by; as cancelledBy) {
              <p class="mt-1 text-sm text-muted-foreground">
                Cancelada por
                <span class="font-medium text-foreground">{{ cancelledBy.name }}</span>
                · {{ cancelledBy.email }}
              </p>
            }
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <span hlmBadge>{{ catalogImportStatusLabel(item.status) }}</span>
            @if (catalogImportIsActive(item.status)) {
              <button hlmBtn variant="destructive" type="button" [disabled]="store.cancelling()" (click)="confirmCancel.set(true)">Cancelar</button>
            }
          </div>
        </header>

        <hlm-card class="space-y-4 p-6">
          <hlm-progress [value]="progress()" class="w-full">
            <hlm-progress-indicator />
          </hlm-progress>
          <p class="text-sm text-muted-foreground">{{ progress() }}%</p>
          <div class="grid gap-3 sm:grid-cols-3">
            <div>
              <p class="text-xs uppercase text-muted-foreground">Creados</p>
              <p class="text-lg font-semibold">{{ item.created_count }}</p>
            </div>
            <div>
              <p class="text-xs uppercase text-muted-foreground">Actualizados</p>
              <p class="text-lg font-semibold">{{ item.updated_count }}</p>
            </div>
            <div>
              <p class="text-xs uppercase text-muted-foreground">Errores</p>
              <p class="text-lg font-semibold">{{ item.failed_count }}</p>
            </div>
          </div>
        </hlm-card>

        @if (item.failure_reason) {
          <div hlmAlert variant="destructive">
            <ng-icon name="lucideCircleAlert" hlmAlertIcon />
            <h4 hlmAlertTitle>El archivo no se pudo procesar</h4>
            <p hlmAlertDescription>{{ item.failure_reason }}</p>
          </div>
        }

        <section class="space-y-3">
          <h2 class="text-lg font-semibold">Errores ({{ store.errorsTotal() || item.failed_count }})</h2>
          <hlm-card class="overflow-hidden p-0">
            @if (store.loadingErrors() && store.errors().length === 0) {
              <div class="flex justify-center py-10"><hlm-spinner class="size-6" /></div>
            } @else {
              <table hlmTable>
                <thead hlmTHead>
                  <tr hlmTr>
                    <th hlmTh>Dónde</th>
                    <th hlmTh>Valores</th>
                    <th hlmTh>Qué</th>
                  </tr>
                </thead>
                <tbody hlmTBody>
                  @for (row of store.errors(); track row.id) {
                    <tr hlmTr>
                      <td hlmTd>
                        Fila {{ row.file_row }} del archivo
                        <span class="block text-xs text-muted-foreground">{{ catalogImportColumnLabel(row.column) }}</span>
                      </td>
                      <td hlmTd class="text-sm text-muted-foreground">{{ row.internal_code || '—' }} · {{ row.name || '—' }} · {{ row.kind || '—' }}</td>
                      <td hlmTd>{{ row.message }}</td>
                    </tr>
                  } @empty {
                    <tr hlmTr>
                      <td hlmTd colspan="3" class="py-10 text-center text-muted-foreground">No se encontraron errores en este proceso.</td>
                    </tr>
                  }
                </tbody>
              </table>
              @if (store.errorsHasMore()) {
                <div class="border-t border-border p-4">
                  <button hlmBtn variant="outline" (click)="store.loadMoreErrors(item.id)" [disabled]="store.loadingMoreErrors()">Cargar más</button>
                </div>
              }
            }
          </hlm-card>
        </section>
      }
    </div>

    <hlm-alert-dialog [state]="confirmCancel() ? 'open' : 'closed'" (closed)="confirmCancel.set(false)">
      <hlm-alert-dialog-content *brnAlertDialogContent>
        <hlm-alert-dialog-header>
          <h2 hlmAlertDialogTitle>¿Cancelar la carga masiva?</h2>
          <p hlmAlertDialogDescription>El lote en curso puede terminar (hasta 5.000 filas). Los ítems ya upsertados no se revierten.</p>
        </hlm-alert-dialog-header>
        <hlm-alert-dialog-footer>
          <button hlmBtn variant="outline" type="button" (click)="confirmCancel.set(false)">Seguir</button>
          <button hlmBtn variant="destructive" type="button" [disabled]="store.cancelling()" (click)="cancel()">Cancelar carga masiva</button>
        </hlm-alert-dialog-footer>
      </hlm-alert-dialog-content>
    </hlm-alert-dialog>
  `,
})
export class ImportDetailPage implements OnInit {
  readonly store = inject(CatalogImportStore);
  readonly catalogImportStatusLabel = catalogImportStatusLabel;
  readonly catalogImportIsActive = catalogImportIsActive;
  readonly catalogImportColumnLabel = catalogImportColumnLabel;
  readonly confirmCancel = signal(false);
  readonly imp = computed(() => this.store.selected());
  readonly progress = computed(() => catalogImportProgressPercent(this.imp()));
  private readonly route = inject(ActivatedRoute);
  private readonly destroyRef = inject(DestroyRef);

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('importId') ?? '';
    this.store.loadImport(id);
    interval(2000)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => {
        const current = this.store.selected();
        if (current && catalogImportIsActive(current.status)) {
          this.store.refreshImport(current.id);
        }
      });
  }

  cancel(): void {
    const current = this.imp();
    if (!current) return;
    this.store.cancel(current.id).subscribe((imp) => {
      if (imp) this.confirmCancel.set(false);
    });
  }
}
