import { DatePipe } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { BrnDialogClose, BrnDialogContent } from '@spartan-ng/brain/dialog';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmDialogImports } from '@spartan-ng/helm/dialog';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { HlmTableImports } from '@spartan-ng/helm/table';
import { FileUploadComponent } from '../../../../core/presentation/components/file-upload';
import { CatalogImportStore } from '../../../application/catalog-import.store';
import { CATALOG_IMPORT_ACCEPT, CATALOG_IMPORT_MAX_FILE_BYTES, catalogImportStatusLabel } from '../../../domain/catalog-import.model';

@Component({
  selector: 'app-catalog-imports',
  standalone: true,
  imports: [
    DatePipe,
    RouterLink,
    NgIcon,
    FileUploadComponent,
    HlmCardImports,
    HlmSpinnerImports,
    HlmTableImports,
    HlmAlertImports,
    HlmBadgeImports,
    HlmButtonImports,
    HlmDialogImports,
    BrnDialogContent,
    BrnDialogClose,
  ],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-5xl space-y-6">
      <header class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <a hlmBtn variant="ghost" size="sm" routerLink=".." class="mb-2 -ms-2">
            <ng-icon name="lucideArrowLeft" />
            Catálogo
          </a>
          <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">Cargas masivas</h1>
          <p class="mt-1 text-sm text-muted-foreground">Historial de cargas masivas de ítems.</p>
        </div>
        <button hlmBtn type="button" (click)="store.openDialog()">
          <ng-icon name="lucideUpload" />
          Cargar
        </button>
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
                <th hlmTh>Solicitante</th>
                <th hlmTh>Creados</th>
                <th hlmTh>Actualizados</th>
                <th hlmTh>Errores</th>
                <th hlmTh>Estado</th>
                <th hlmTh>Fecha</th>
              </tr>
            </thead>
            <tbody hlmTBody>
              @for (imp of store.imports(); track imp.id) {
                <tr hlmTr class="cursor-pointer hover:bg-muted/40" (click)="openDetail(imp.id)">
                  <td hlmTd>
                    <p class="font-medium">{{ imp.requested_by.name }}</p>
                    <p class="text-xs text-muted-foreground">{{ imp.requested_by.email }}</p>
                  </td>
                  <td hlmTd>{{ imp.created_count }}</td>
                  <td hlmTd>{{ imp.updated_count }}</td>
                  <td hlmTd>{{ imp.failed_count }}</td>
                  <td hlmTd>
                    <span hlmBadge variant="secondary">{{ catalogImportStatusLabel(imp.status) }}</span>
                  </td>
                  <td hlmTd>{{ imp.created_at | date: 'short' }}</td>
                </tr>
              } @empty {
                <tr hlmTr>
                  <td hlmTd colspan="6" class="py-10 text-center text-muted-foreground">Aún no hay cargas masivas.</td>
                </tr>
              }
            </tbody>
          </table>
          @if (store.hasMore()) {
            <div class="border-t border-border p-4">
              <button hlmBtn variant="outline" (click)="store.loadMoreImports()" [disabled]="store.loadingMore()">Cargar más</button>
            </div>
          }
        }
      </hlm-card>
    </div>

    <hlm-dialog [state]="store.dialogOpen() ? 'open' : 'closed'" (closed)="store.closeDialog()">
      <hlm-dialog-content *brnDialogContent class="sm:max-w-lg">
        <hlm-dialog-header>
          <h2 hlmDialogTitle>Cargar catálogo</h2>
        </hlm-dialog-header>
        <div class="space-y-4">
          <app-file-upload
            [accept]="accept"
            [multiple]="false"
            [maxFiles]="1"
            [maxFileSizeBytes]="maxBytes"
            [validateFile]="store.validateCsv"
            [items]="store.uploadQueue()"
            [isPickerDisabled]="store.submitting()"
            dropzoneTitle="Arrastra un CSV o selecciónalo"
            dropzoneDescription="UTF-8, columnas internal_code, name, kind. Máximo 500 MiB."
            (filesSelected)="onFiles($event)"
            (removeRequested)="store.removeFile()"
            (clearAllRequested)="store.removeFile()"
          />
          <button type="button" hlmBtn variant="ghost" size="sm" (click)="store.downloadTemplate()">Descargar plantilla</button>
          @if (store.errorMessage(); as err) {
            <p class="text-sm text-destructive">{{ err }}</p>
          }
        </div>
        <hlm-dialog-footer>
          <button hlmBtn variant="outline" brnDialogClose [disabled]="store.submitting()">Cancelar</button>
          <button hlmBtn [disabled]="!store.canSubmitImport()" (click)="submitImport()">
            {{ store.submitting() ? 'Encolando…' : 'Cargar' }}
          </button>
        </hlm-dialog-footer>
      </hlm-dialog-content>
    </hlm-dialog>
  `,
})
export class ImportsPage implements OnInit {
  readonly store = inject(CatalogImportStore);
  readonly catalogImportStatusLabel = catalogImportStatusLabel;
  readonly accept = CATALOG_IMPORT_ACCEPT;
  readonly maxBytes = CATALOG_IMPORT_MAX_FILE_BYTES;
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  ngOnInit(): void {
    this.store.loadImports();
  }

  openDetail(id: string): void {
    void this.router.navigate([id], { relativeTo: this.route });
  }

  onFiles(files: File[]): void {
    const file = files[0];
    if (file) this.store.addFile(file);
  }

  submitImport(): void {
    this.store.queueImport().subscribe((imp) => {
      if (imp) void this.router.navigate([imp.id], { relativeTo: this.route });
    });
  }
}
