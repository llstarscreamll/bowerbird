import { CommonModule } from '@angular/common';
import { Component, computed, inject } from '@angular/core';
import { InvoiceHistoryImportStore } from '../../../application/invoice-history-import.store';
import { ModalComponent } from '../../../../core/presentation/components/modal/modal.component';
import { FileUploadComponent, FileUploadQueueItem } from '../../../../core/presentation/components/file-upload/file-upload.component';
import { INVOICE_HISTORY_ACCEPT } from '../../../domain/invoice-history-import.model';

@Component({
  selector: 'app-master-invoices',
  standalone: true,
  imports: [CommonModule, ModalComponent, FileUploadComponent],
  host: {
    class: 'flex-1 flex flex-col min-h-0 w-full',
  },
  template: `
    <div class="h-full w-full bg-slate-50 dark:bg-slate-950 p-8 overflow-y-auto transition-colors duration-200 flex-1 flex flex-col">
      <div class="mx-auto space-y-6 w-full">
        <!-- Header -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h2 class="text-2xl font-bold leading-7 text-slate-900 dark:text-white sm:truncate sm:text-3xl sm:tracking-tight">Facturas</h2>
            <p class="mt-1 text-sm leading-6 text-slate-500 dark:text-slate-400">Gestiona, filtra y revisa todas tus facturas electrónicas.</p>
          </div>
          <div class="flex items-center gap-3">
            <button class="btn-secondary">
              <span class="material-icons-outlined text-[18px] mr-1.5">filter_list</span>
              Filtrar
            </button>
            <button class="btn-primary">
              <span class="material-icons-outlined text-[18px] mr-1.5">add</span>
              Nueva Factura
            </button>
          </div>
        </div>

        <!-- Empty State Master -->
        <div class="card flex flex-col items-center justify-center py-20 text-center shadow-sm">
          <div class="w-20 h-20 bg-slate-100 dark:bg-slate-800 rounded-full flex items-center justify-center mb-6 shadow-[inset_0_2px_4px_rgba(0,0,0,0.02)] transition-colors">
            <span class="material-icons-outlined text-slate-300 dark:text-slate-600 text-4xl">receipt_long</span>
          </div>
          <h3 class="text-lg font-medium text-slate-900 dark:text-white">Aún no hay facturas</h3>
          <p class="mt-2 text-sm text-slate-500 dark:text-slate-400 max-w-sm mb-6">
            No se han encontrado facturas en este entorno. Pronto podrás sincronizarlas desde tu bandeja o crearlas manualmente.
          </p>

          <button class="btn-secondary" [disabled]="isUploading() || isAnalyzing()" (click)="openImportModal()">
            <span class="material-icons-outlined text-[18px] mr-1.5">cloud_download</span>
            {{ isUploading() ? 'Importando...' : 'Importar histórico' }}
          </button>

          <p *ngIf="errorMessage()" class="mt-3 text-sm text-rose-600 dark:text-rose-300">{{ errorMessage() }}</p>
        </div>
      </div>
    </div>

    <app-modal [isOpen]="isImportModalOpen()" title="Importar historico de facturas" [showDefaultFooter]="false" (close)="closeImportModal()">
      <div class="space-y-4">
        <app-file-upload
          [accept]="historyImportAccept"
          [items]="uploadQueueItems()"
          [isPickerDisabled]="isUploading() || isAnalyzing()"
          [disableActions]="isAnalyzing()"
          (filesSelected)="onFilesAdded($event)"
          (cancelRequested)="cancelUpload($event)"
          (removeRequested)="removeFile($event)"
        />

        <p *ngIf="errorMessage()" class="text-sm text-rose-600 dark:text-rose-300">{{ errorMessage() }}</p>

        <div class="rounded-xl border border-sky-200 bg-sky-50 p-3 mt-4 text-sm text-sky-800 dark:border-sky-800/70 dark:bg-sky-950/30 dark:text-sky-100">
          <p class="font-medium">El análisis se ejecuta en segundo plano.</p>
          <p class="mt-1">Cuando presiones Analizar, enviaremos todos los archivos cargados y el proceso continuará de forma asíncrona.</p>
        </div>
      </div>

      <ng-container modal-footer>
        <button type="button" class="btn-primary w-full sm:w-auto" [disabled]="!canAnalyze()" (click)="analyzeFiles()">
          <span class="material-icons-outlined text-[18px] mr-1.5">auto_awesome</span>
          {{ isAnalyzing() ? 'Encolando...' : 'Analizar' }}
        </button>
        <button type="button" class="btn-secondary mt-3 w-full sm:mt-0 sm:w-auto" [disabled]="isUploading() || isAnalyzing()" (click)="closeImportModal()">Cancelar</button>
      </ng-container>
    </app-modal>
  `,
})
export class MasterInvoicesComponent {
  private readonly importStore = inject(InvoiceHistoryImportStore);

  readonly historyImportAccept = INVOICE_HISTORY_ACCEPT;
  readonly isImportModalOpen = this.importStore.isImportModalOpen;
  readonly isUploading = this.importStore.uploading;
  readonly isAnalyzing = this.importStore.analyzing;
  readonly errorMessage = this.importStore.errorMessage;
  readonly queuedFiles = this.importStore.queuedFiles;
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

  openImportModal(): void {
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

  analyzeFiles(): void {
    this.importStore.analyzeUploadedFiles();
  }
}
