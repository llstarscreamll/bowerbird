import { CommonModule, CurrencyPipe, DatePipe } from '@angular/common';
import { Component, OnInit, computed, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { BrnDialogClose, BrnDialogContent } from '@spartan-ng/brain/dialog';
import { InvoiceHistoryImportStore } from '../../../application/invoice-history-import.store';
import { InvoicesStore } from '../../../application/invoices.store';
import { FileUploadComponent, FileUploadQueueItem } from '../../../../core/presentation/components/file-upload';
import { generateUlid } from '../../../../core/utils/ulid';
import { INVOICE_HISTORY_ACCEPT, INVOICE_HISTORY_MAX_FILE_SIZE_BYTES, supportsInvoiceHistoryFile } from '../../../domain/invoice-history-import.model';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmDialogImports } from '@spartan-ng/helm/dialog';
import { HlmEmptyImports } from '@spartan-ng/helm/empty';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { HlmTableImports } from '@spartan-ng/helm/table';

@Component({
  selector: 'app-invoices-master',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    NgIcon,
    FileUploadComponent,
    CurrencyPipe,
    DatePipe,
    RouterLink,
    HlmCardImports,
    HlmButtonImports,
    HlmDialogImports,
    HlmSpinnerImports,
    HlmEmptyImports,
    HlmTableImports,
    HlmAlertImports,
    HlmBadgeImports,
    BrnDialogContent,
    BrnDialogClose,
  ],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full' },
  template: `
    <div class="h-full w-full flex-1 overflow-y-auto p-8">
      <div class="mx-auto w-full space-y-6">
        <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
          <div>
            <h2 class="text-2xl font-bold tracking-tight sm:text-3xl">Facturas</h2>
            <p class="mt-1 text-sm text-muted-foreground">Gestiona tus facturas electrónicas.</p>
          </div>
          <div class="flex items-center gap-3">
            @if (hasInvoices()) {
              <button hlmBtn variant="outline" (click)="openImportModal()">
                <ng-icon name="lucideCloudDownload" />
                Importar
              </button>
            }
            <button hlmBtn variant="outline">
              <ng-icon name="lucideFilter" />
              Filtrar
            </button>
            <button hlmBtn>
              <ng-icon name="lucidePlus" />
              Nueva Factura
            </button>
          </div>
        </div>

        @if (isLoading() && !hasInvoices()) {
          <div class="flex items-center justify-center py-20">
            <hlm-spinner class="size-8 text-primary" />
          </div>
        }

        @if (!isLoading() && !hasInvoices()) {
          <hlm-empty class="py-20">
            <ng-icon hlm name="lucideReceipt" class="text-4xl text-muted-foreground/40" />
            <h3 hlmEmptyTitle>Aún no hay facturas</h3>
            <p hlmEmptyDescription>No se han encontrado facturas en este entorno. Pronto podrás sincronizarlas desde tu bandeja o crearlas manualmente.</p>
            <button hlmBtn variant="outline" class="mt-6" [disabled]="isUploading() || isAnalyzing()" (click)="openImportModal()">
              <ng-icon name="lucideCloudDownload" />
              {{ isUploading() ? 'Importando...' : 'Importar histórico' }}
            </button>
            @if (errorMessage()) {
              <p class="mt-3 text-sm text-destructive">{{ errorMessage() }}</p>
            }
          </hlm-empty>
        }

        @if (hasInvoices()) {
          <hlm-card class="overflow-hidden p-0">
            <div class="overflow-x-auto">
              <table hlmTable>
                <thead hlmTHead>
                  <tr hlmTr>
                    <th hlmTh>Número</th>
                    <th hlmTh>Emisor</th>
                    <th hlmTh>Catálogo</th>
                    <th hlmTh>Total</th>
                    <th hlmTh class="text-right">Fecha Emisión</th>
                  </tr>
                </thead>
                <tbody hlmTBody>
                  @for (invoice of invoices(); track invoice.id) {
                    <tr hlmTr>
                      <td hlmTd class="font-medium">
                        <a [routerLink]="[invoice.id]" class="text-primary hover:underline">{{ invoice.invoice_number || 'N/A' }}</a>
                      </td>
                      <td hlmTd>
                        <div class="max-w-[200px] truncate font-medium" [title]="invoice.issuer_name">{{ invoice.issuer_name || 'Desconocido' }}</div>
                        <div class="text-xs text-muted-foreground">{{ invoice.issuer_tax_id }}</div>
                      </td>
                      <td hlmTd>
                        <span hlmBadge [variant]="linkingBadgeVariant(invoice.linking_status)">{{ linkingStatusLabel(invoice.linking_status) }}</span>
                      </td>
                      <td hlmTd class="font-medium">{{ invoice.grand_total | currency: invoice.currency_code : 'symbol' : '1.2-2' }}</td>
                      <td hlmTd class="text-right text-muted-foreground">{{ invoice.issue_date | date: 'mediumDate' }}</td>
                    </tr>
                  }
                </tbody>
              </table>
            </div>
            <div class="flex items-center justify-center border-t px-4 py-3 sm:px-6">
              @if (hasMore()) {
                <button hlmBtn variant="outline" (click)="loadMore()" [disabled]="isLoadingMore()">
                  {{ isLoadingMore() ? 'Cargando...' : 'Cargar más' }}
                </button>
              } @else if (hasInvoices()) {
                <p class="text-sm text-muted-foreground">Has llegado al final de la lista.</p>
              }
            </div>
          </hlm-card>
        }
      </div>
    </div>

    <hlm-dialog [state]="isImportModalOpen() ? 'open' : 'closed'" (closed)="closeImportModal()">
      <hlm-dialog-content *brnDialogContent class="sm:max-w-lg">
        <hlm-dialog-header>
          <h2 hlmDialogTitle>Importar historico de facturas</h2>
        </hlm-dialog-header>
        <div class="space-y-4">
          <app-file-upload
            [accept]="historyImportAccept"
            [validateFile]="validateHistoryFile"
            [maxFileSizeBytes]="historyMaxFileSizeBytes"
            [items]="uploadQueueItems()"
            [isPickerDisabled]="isUploading() || isAnalyzing()"
            [disableActions]="isAnalyzing()"
            (filesSelected)="onFilesAdded($event)"
            (cancelRequested)="cancelUpload($event)"
            (removeRequested)="removeFile($event)"
            (retryRequested)="retryUpload($event)"
            (clearAllRequested)="clearAllFiles()"
          />
          @if (errorMessage()) {
            <p class="text-sm text-destructive">{{ errorMessage() }}</p>
          }
          <hlm-alert>
            <ng-icon hlm name="lucideInfo" />
            <h4 hlmAlertTitle>El análisis se ejecuta en segundo plano.</h4>
            <p hlmAlertDescription>Cuando presiones Analizar, enviaremos todos los archivos cargados y el proceso continuará de forma asíncrona.</p>
          </hlm-alert>
        </div>
        <hlm-dialog-footer>
          <button hlmBtn variant="outline" brnDialogClose [disabled]="isUploading() || isAnalyzing()">Cancelar</button>
          <button hlmBtn [disabled]="!canAnalyze()" (click)="analyzeFiles()">
            <ng-icon name="lucideSparkles" />
            {{ isAnalyzing() ? 'Encolando...' : 'Analizar' }}
          </button>
        </hlm-dialog-footer>
      </hlm-dialog-content>
    </hlm-dialog>
  `,
})
export class MasterPage implements OnInit {
  private readonly importStore = inject(InvoiceHistoryImportStore);
  private readonly invoicesStore = inject(InvoicesStore);
  private readonly formBuilder = inject(FormBuilder);

  readonly historyImportAccept = INVOICE_HISTORY_ACCEPT;
  readonly historyMaxFileSizeBytes = INVOICE_HISTORY_MAX_FILE_SIZE_BYTES;
  readonly validateHistoryFile = supportsInvoiceHistoryFile;
  readonly isImportModalOpen = this.importStore.isImportModalOpen;
  readonly isUploading = this.importStore.uploading;
  readonly isAnalyzing = this.importStore.analyzing;
  readonly errorMessage = this.importStore.errorMessage;
  readonly queuedFiles = this.importStore.queuedFiles;
  readonly invoices = this.invoicesStore.invoices;
  readonly hasInvoices = this.invoicesStore.hasInvoices;
  readonly isLoading = this.invoicesStore.isLoading;
  readonly isLoadingMore = this.invoicesStore.isLoadingMore;
  readonly hasMore = this.invoicesStore.hasMore;
  readonly limit = this.invoicesStore.limit;

  readonly importForm = this.formBuilder.nonNullable.group({
    id: ['', [Validators.required, Validators.pattern(/^[0-9A-HJKMNPQRSTVWXYZ]{26}$/)]],
  });
  readonly uploadQueueItems = computed<FileUploadQueueItem[]>(() =>
    this.queuedFiles().map((queued) => ({
      id: queued.id,
      name: queued.file.name,
      size: queued.file.size,
      status: queued.status,
      progress: queued.uploadProgress,
    })),
  );
  readonly canAnalyze = this.importStore.canAnalyze;

  constructor() {
    this.resetRequestId();
  }

  ngOnInit(): void {
    this.invoicesStore.loadInvoices();
  }

  loadMore(): void {
    this.invoicesStore.loadMore();
  }

  openImportModal(): void {
    this.resetRequestId();
    this.importStore.openImportModal();
  }

  closeImportModal(): void {
    this.importStore.closeImportModal();
  }

  onFilesAdded(files: File[]): void {
    this.importStore.addFiles(files);
  }

  removeFile(fileId: string): void {
    this.importStore.removeFile(fileId);
  }

  cancelUpload(fileId: string): void {
    this.importStore.cancelFileUpload(fileId);
  }

  retryUpload(fileId: string): void {
    this.importStore.retryFileUpload(fileId);
  }

  clearAllFiles(): void {
    this.importStore.clearAllFiles();
  }

  analyzeFiles(): void {
    const id = this.importForm.controls.id.value;
    if (!this.importForm.valid || !id) return;
    this.importStore.analyzeUploadedFiles(id);
  }

  linkingStatusLabel(status?: string): string {
    switch (status) {
      case 'linked':
        return 'Vinculado';
      case 'failed':
        return 'Fallido';
      case 'pending':
        return 'Pendiente';
      default:
        return status || 'Pendiente';
    }
  }

  linkingBadgeVariant(status?: string): 'default' | 'secondary' | 'destructive' | 'outline' {
    if (status === 'failed') return 'destructive';
    if (status === 'linked') return 'default';
    return 'secondary';
  }

  private resetRequestId(): void {
    this.importForm.patchValue({ id: generateUlid() });
  }
}
