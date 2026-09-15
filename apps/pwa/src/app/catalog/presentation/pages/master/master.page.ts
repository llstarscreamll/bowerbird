import { CommonModule } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
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
import { CatalogStore } from '../../../application/catalog.store';
import { CatalogImportStore } from '../../../application/catalog-import.store';
import { creationSourceLabel } from '../../../domain/catalog.model';
import { CATALOG_IMPORT_ACCEPT, CATALOG_IMPORT_MAX_FILE_BYTES, catalogImportIsActive, catalogImportStatusLabel } from '../../../domain/catalog-import.model';

@Component({
  selector: 'app-catalog-master',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
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
          <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">Catálogo</h1>
          <p class="mt-1 text-sm text-muted-foreground">Ítems (productos, servicios, activos) vinculados desde líneas de factura.</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <a hlmBtn variant="outline" routerLink="imports">
            <ng-icon name="lucideList" />
            Importaciones
          </a>
          <button hlmBtn variant="outline" type="button" (click)="imports.openDialog()">
            <ng-icon name="lucideUpload" />
            Importar
          </button>
          <a hlmBtn routerLink="new">
            <ng-icon name="lucidePlus" />
            Nuevo
          </a>
        </div>
      </header>

      @if (imports.activeImport(); as active) {
        <a class="block" [routerLink]="['imports', active.id]">
          <div hlmAlert>
            <ng-icon name="lucideInfo" hlmAlertIcon />
            <h4 hlmAlertTitle>Importación {{ catalogImportStatusLabel(active.status) }}</h4>
            <p hlmAlertDescription>Hay un proceso {{ catalogImportIsActive(active.status) ? 'en curso' : active.status }}. Ver monitor.</p>
          </div>
        </a>
      }

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      <div class="flex flex-col gap-2 sm:flex-row">
        <input
          class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          placeholder="Buscar por nombre o código…"
          [ngModel]="store.search()"
          (ngModelChange)="store.search.set($event)"
          (keyup.enter)="store.loadItems()"
        />
        <button hlmBtn type="button" variant="outline" (click)="store.loadItems()">
          <ng-icon name="lucideSearch" />
          Buscar
        </button>
      </div>

      <hlm-card class="overflow-hidden p-0">
        @if (store.loading()) {
          <div class="flex justify-center py-16"><hlm-spinner class="size-8" /></div>
        } @else {
          <table hlmTable>
            <thead hlmTHead>
              <tr hlmTr>
                <th hlmTh>Nombre</th>
                <th hlmTh>Código interno</th>
                <th hlmTh>Tipo</th>
                <th hlmTh>Origen</th>
                <th hlmTh>Estado</th>
              </tr>
            </thead>
            <tbody hlmTBody>
              @for (item of store.items(); track item.id) {
                <tr hlmTr class="cursor-pointer hover:bg-muted/40" (click)="openDetail(item.id)">
                  <td hlmTd class="font-medium">{{ item.name }}</td>
                  <td hlmTd class="text-muted-foreground">{{ item.internal_code || '—' }}</td>
                  <td hlmTd>
                    <span hlmBadge variant="secondary">{{ item.kind }}</span>
                  </td>
                  <td hlmTd>{{ creationSourceLabel(item.creation_source) }}</td>
                  <td hlmTd>{{ item.status }}</td>
                </tr>
              } @empty {
                <tr hlmTr>
                  <td hlmTd colspan="5" class="py-10 text-center text-muted-foreground">Aún no hay ítems en el catálogo.</td>
                </tr>
              }
            </tbody>
          </table>
          @if (store.hasMore()) {
            <div class="border-t border-border p-4">
              <button hlmBtn variant="outline" (click)="store.loadMore()" [disabled]="store.loadingMore()">
                @if (store.loadingMore()) {
                  <hlm-spinner class="size-4" />
                }
                Cargar más
              </button>
            </div>
          }
        }
      </hlm-card>
    </div>

    <hlm-dialog [state]="imports.dialogOpen() ? 'open' : 'closed'" (closed)="imports.closeDialog()">
      <hlm-dialog-content *brnDialogContent class="sm:max-w-lg">
        <hlm-dialog-header>
          <h2 hlmDialogTitle>Importar catálogo</h2>
        </hlm-dialog-header>
        <div class="space-y-4">
          <app-file-upload
            [accept]="accept"
            [multiple]="false"
            [maxFiles]="1"
            [maxFileSizeBytes]="maxBytes"
            [validateFile]="imports.validateCsv"
            [items]="imports.uploadQueue()"
            [isPickerDisabled]="imports.submitting()"
            dropzoneTitle="Arrastra un CSV o selecciónalo"
            dropzoneDescription="UTF-8, columnas internal_code, name, kind. Máximo 500 MiB."
            (filesSelected)="onFiles($event)"
            (removeRequested)="imports.removeFile()"
            (clearAllRequested)="imports.removeFile()"
          />
          <button type="button" hlmBtn variant="ghost" size="sm" (click)="imports.downloadTemplate()">Descargar plantilla</button>
          @if (imports.errorMessage(); as err) {
            <p class="text-sm text-destructive">{{ err }}</p>
          }
        </div>
        <hlm-dialog-footer>
          <button hlmBtn variant="outline" brnDialogClose [disabled]="imports.submitting()">Cancelar</button>
          <button hlmBtn [disabled]="!imports.canSubmitImport()" (click)="submitImport()">
            {{ imports.submitting() ? 'Encolando…' : 'Importar' }}
          </button>
        </hlm-dialog-footer>
      </hlm-dialog-content>
    </hlm-dialog>
  `,
})
export class MasterPage implements OnInit {
  readonly store = inject(CatalogStore);
  readonly imports = inject(CatalogImportStore);
  readonly creationSourceLabel = creationSourceLabel;
  readonly catalogImportStatusLabel = catalogImportStatusLabel;
  readonly catalogImportIsActive = catalogImportIsActive;
  readonly accept = CATALOG_IMPORT_ACCEPT;
  readonly maxBytes = CATALOG_IMPORT_MAX_FILE_BYTES;
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  ngOnInit(): void {
    this.store.loadItems();
    this.imports.loadImports();
  }

  openDetail(id: string): void {
    void this.router.navigate([id], { relativeTo: this.route });
  }

  onFiles(files: File[]): void {
    const file = files[0];
    if (file) this.imports.addFile(file);
  }

  submitImport(): void {
    this.imports.queueImport().subscribe((imp) => {
      if (imp) void this.router.navigate(['imports', imp.id], { relativeTo: this.route });
    });
  }
}
