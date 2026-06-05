import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { InvoiceHistoryImportStore, InvoiceHistoryQueuedFile } from '../../../application/invoice-history-import.store';
import { ModalComponent } from '../../../../core/presentation/components/modal/modal.component';
import { INVOICE_HISTORY_ACCEPT } from '../../../domain/invoice-history-import.model';

@Component({
  selector: 'app-invoices-list',
  standalone: true,
  imports: [CommonModule, ModalComponent],
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
        <input #historyFileInput type="file" class="hidden" [accept]="historyImportAccept" multiple (change)="onFilesSelected($event)" />

        <div
          class="rounded-2xl border border-dashed p-6 sm:p-8 text-center transition-colors"
          [class.border-indigo-400]="isDragOver"
          [class.bg-indigo-50]="isDragOver"
          [class.dark:bg-indigo-950/20]="isDragOver"
          [class.border-slate-300]="!isDragOver"
          [class.bg-slate-100/70]="!isDragOver"
          [class.dark:border-slate-700]="!isDragOver"
          [class.dark:bg-slate-800/60]="!isDragOver"
          (dragover)="onDragOver($event)"
          (dragleave)="onDragLeave($event)"
          (drop)="onDrop($event)"
        >
          <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-white text-slate-500 shadow-sm dark:bg-slate-900 dark:text-slate-300">
            <span class="material-icons-outlined text-3xl">upload</span>
          </div>
          <p class="text-base font-semibold text-slate-900 dark:text-slate-100">Arrastra tus archivos aqui o selecciona</p>
          <p class="mt-1 text-sm text-slate-500 dark:text-slate-300">XML, PDF o ZIP - maximo 1 GB por archivo.</p>
          <button type="button" class="btn-secondary mt-4" [disabled]="isUploading() || isAnalyzing()" (click)="openFilePicker(historyFileInput)">
            <span class="material-icons-outlined text-[18px] mr-1.5">folder_open</span>
            Buscar archivos
          </button>
        </div>

        <div *ngIf="uploadedQueueFiles().length > 0" class="space-y-2">
          <div class="flex items-center justify-between px-1">
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-300">Subidos</p>
            <p class="text-xs text-slate-500 dark:text-slate-300">{{ uploadedQueueFiles().length }} archivo(s)</p>
          </div>
          <div class="space-y-2 max-h-48 overflow-y-auto pr-1">
            <div *ngFor="let queued of uploadedQueueFiles(); trackBy: trackByFileId" class="rounded-xl border border-slate-200 bg-white px-3 py-3 dark:border-slate-700 dark:bg-slate-900/70">
              <div class="flex items-center gap-3">
                <div
                  class="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:bg-slate-800 dark:text-slate-200"
                >
                  {{ fileExtension(queued.file.name) }}
                </div>
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-semibold text-slate-900 dark:text-slate-100">{{ queued.file.name }}</p>
                  <p class="text-sm text-slate-500 dark:text-slate-300">{{ formatBytes(queued.file.size) }} - Listo para analizar</p>
                </div>
                <button
                  type="button"
                  class="rounded-lg p-1.5 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-800 disabled:cursor-not-allowed disabled:opacity-40 dark:text-slate-300 dark:hover:bg-slate-800 dark:hover:text-slate-100"
                  [disabled]="isUploading() || isAnalyzing()"
                  (click)="removeFile(queued.id)"
                  [attr.aria-label]="'Eliminar ' + queued.file.name"
                >
                  <span class="material-icons-outlined text-[18px]">delete</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div *ngIf="uploadingFiles().length > 0" class="space-y-2 border-t border-slate-200 pt-3 dark:border-slate-700/80">
          <div class="flex items-center justify-between px-1">
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-300">Subiendo</p>
            <p class="text-xs text-slate-500 dark:text-slate-300">{{ uploadProgressPercent() }}%</p>
          </div>
          <div class="space-y-2 max-h-40 overflow-y-auto pr-1">
            <div *ngFor="let queued of uploadingFiles(); trackBy: trackByFileId" class="rounded-xl border border-slate-200 bg-white px-3 py-3 dark:border-slate-700 dark:bg-slate-900/70">
              <div class="flex items-center gap-3">
                <div
                  class="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:bg-slate-800 dark:text-slate-200"
                >
                  {{ fileExtension(queued.file.name) }}
                </div>
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-semibold text-slate-900 dark:text-slate-100">{{ queued.file.name }}</p>
                  <p class="text-sm text-slate-500 dark:text-slate-300">{{ formatBytes(queued.file.size) }} - {{ statusLabel(queued) }}</p>
                </div>
              </div>
              <div class="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700">
                <div
                  class="h-full rounded-full"
                  [class.w-full]="queued.status === 'pending'"
                  [class.bg-rose-500]="queued.status === 'failed'"
                  [class.animate-pulse]="queued.status === 'uploading'"
                  [class.w-1/2]="queued.status === 'uploading'"
                  [class.bg-indigo-600]="queued.status !== 'failed'"
                ></div>
              </div>
            </div>
          </div>
        </div>

        <div *ngIf="queuedFiles().length === 0" class="rounded-xl border border-slate-200 bg-slate-50 p-3 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/50 dark:text-slate-300">
          Aun no has seleccionado archivos.
        </div>

        <p *ngIf="errorMessage()" class="text-sm text-rose-600 dark:text-rose-300">{{ errorMessage() }}</p>

        <div class="rounded-xl border border-sky-200 bg-sky-50 p-3 text-sm text-sky-800 dark:border-sky-800/70 dark:bg-sky-950/30 dark:text-sky-100">
          <p class="font-medium">El analisis se ejecuta en segundo plano.</p>
          <p class="mt-1">Cuando presiones Analizar, enviaremos todos los archivos cargados y el proceso continuara de forma asincrona.</p>
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
export class InvoicesListComponent {
  private readonly importStore = inject(InvoiceHistoryImportStore);

  readonly historyImportAccept = INVOICE_HISTORY_ACCEPT;
  readonly isImportModalOpen = this.importStore.isImportModalOpen;
  readonly isUploading = this.importStore.uploading;
  readonly isAnalyzing = this.importStore.analyzing;
  readonly errorMessage = this.importStore.errorMessage;
  readonly queuedFiles = this.importStore.queuedFiles;
  readonly uploadingFiles = this.importStore.uploadingFiles;
  readonly uploadedQueueFiles = this.importStore.uploadedQueueFiles;
  readonly uploadProgressPercent = this.importStore.uploadProgressPercent;
  readonly canAnalyze = this.importStore.canAnalyze;

  isDragOver = false;

  openImportModal(): void {
    this.importStore.openImportModal();
  }

  closeImportModal(): void {
    this.isDragOver = false;
    this.importStore.closeImportModal();
  }

  openFilePicker(input: HTMLInputElement): void {
    input.click();
  }

  onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const files = input.files ? Array.from(input.files) : [];

    this.importStore.addFiles(files);
    input.value = '';
  }

  onDragOver(event: DragEvent): void {
    event.preventDefault();
    this.isDragOver = true;
  }

  onDragLeave(event: DragEvent): void {
    event.preventDefault();
    this.isDragOver = false;
  }

  onDrop(event: DragEvent): void {
    event.preventDefault();
    this.isDragOver = false;

    const files = event.dataTransfer?.files ? Array.from(event.dataTransfer.files) : [];
    this.importStore.addFiles(files);
  }

  removeFile(fileId: string): void {
    this.importStore.removeFile(fileId);
  }

  analyzeFiles(): void {
    this.importStore.analyzeUploadedFiles();
  }

  trackByFileId(_: number, item: InvoiceHistoryQueuedFile): string {
    return item.id;
  }

  fileExtension(name: string): string {
    return name.split('.').pop()?.toUpperCase() ?? 'FILE';
  }

  statusLabel(queued: InvoiceHistoryQueuedFile): string {
    if (queued.status === 'uploaded') {
      return 'Listo para analizar';
    }
    if (queued.status === 'failed') {
      return 'Error al subir, intenta de nuevo';
    }
    if (queued.status === 'uploading') {
      return 'Subiendo archivo';
    }
    if (queued.status === 'pending') {
      return 'En cola';
    }

    return 'Error al subir';
  }

  formatBytes(bytes: number): string {
    if (bytes < 1024) {
      return `${bytes} B`;
    }

    const units = ['KB', 'MB', 'GB'];
    let size = bytes / 1024;
    let unitIndex = 0;

    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex += 1;
    }

    return `${size.toFixed(1)} ${units[unitIndex]}`;
  }
}
