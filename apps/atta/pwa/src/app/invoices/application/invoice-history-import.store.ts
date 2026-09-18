import { Injectable, computed, inject, signal } from '@angular/core';
import { finalize, Subscription } from 'rxjs';
import { FileReference } from '../../core/domain/file-storage.model';
import { FileStorageService } from '../../core/services/file-storage.service';
import { ToastService } from '../../core/services/toast.service';
import { InvoiceHistoryAnalyzeFileReference } from '../domain/invoice-history-analysis.model';
import { InvoiceHistoryHttpService } from '../infrastructure/invoice-history.http.service';

type InvoiceHistoryQueueStatus = 'pending' | 'uploading' | 'uploaded' | 'failed';

export interface InvoiceHistoryQueuedFile {
  id: string;
  file: File;
  status: InvoiceHistoryQueueStatus;
  uploadProgress: number;
  uploadedReference: InvoiceHistoryAnalyzeFileReference | null;
}

interface UploadBatchState {
  remaining: number;
  uploadedCount: number;
  failedCount: number;
}

@Injectable({ providedIn: 'root' })
export class InvoiceHistoryImportStore {
  readonly isImportModalOpen = signal(false);
  readonly uploading = signal(false);
  readonly analyzing = signal(false);
  readonly errorMessage = signal('');
  readonly queuedFiles = signal<InvoiceHistoryQueuedFile[]>([]);

  readonly uploadedFiles = computed(() =>
    this.queuedFiles()
      .filter((entry) => entry.status === 'uploaded' && entry.uploadedReference)
      .map((entry) => entry.uploadedReference as InvoiceHistoryAnalyzeFileReference),
  );
  readonly uploadingFiles = computed(() => this.queuedFiles().filter((entry) => entry.status === 'uploading' || entry.status === 'pending'));
  readonly uploadedQueueFiles = computed(() => this.queuedFiles().filter((entry) => entry.status === 'uploaded'));
  readonly uploadProgressPercent = computed(() => {
    const files = this.queuedFiles();
    if (files.length === 0) {
      return 0;
    }

    const uploadedCount = files.filter((entry) => entry.status === 'uploaded').length;
    return Math.round((uploadedCount / files.length) * 100);
  });

  readonly hasPendingFiles = computed(() => this.queuedFiles().some((entry) => entry.status === 'pending'));
  readonly canAnalyze = computed(() => {
    const files = this.queuedFiles();
    return !this.uploading() && !this.analyzing() && files.length > 0 && files.every((entry) => entry.status === 'uploaded');
  });

  private readonly fileStorage = inject(FileStorageService);
  private readonly invoiceHistoryHttp = inject(InvoiceHistoryHttpService);
  private readonly toast = inject(ToastService);
  private readonly activeUploads = new Map<string, Subscription>();
  private currentBatch: UploadBatchState | null = null;
  private nextId = 1;

  openImportModal(): void {
    this.isImportModalOpen.set(true);
    this.errorMessage.set('');
  }

  closeImportModal(): void {
    if (this.uploading() || this.analyzing()) {
      return;
    }

    this.isImportModalOpen.set(false);
    this.resetState();
  }

  addFiles(files: readonly File[]): void {
    if (files.length === 0 || this.analyzing()) {
      return;
    }

    this.errorMessage.set('');

    const newEntries = files.map((file) => ({
      id: String(this.nextId++),
      file,
      status: 'pending' as const,
      uploadProgress: 0,
      uploadedReference: null,
    }));

    this.queuedFiles.update((entries) => [...entries, ...newEntries]);

    if (!this.uploading()) {
      this.uploadPendingFiles();
    }
  }

  removeFile(fileId: string): void {
    if (this.analyzing()) {
      return;
    }

    const entry = this.queuedFiles().find((item) => item.id === fileId);
    if (!entry) {
      return;
    }

    if (entry.status === 'uploading' || entry.status === 'pending') {
      this.cancelFileUpload(fileId);
      return;
    }

    this.queuedFiles.update((entries) => entries.filter((entry) => entry.id !== fileId));
    this.errorMessage.set('');
  }

  clearAllFiles(): void {
    if (this.analyzing()) {
      return;
    }

    for (const subscription of this.activeUploads.values()) {
      subscription.unsubscribe();
    }
    this.activeUploads.clear();
    this.currentBatch = null;
    this.uploading.set(false);
    this.queuedFiles.set([]);
    this.errorMessage.set('');
  }

  retryFileUpload(fileId: string): void {
    if (this.analyzing()) {
      return;
    }

    const entry = this.queuedFiles().find((item) => item.id === fileId);
    if (!entry || entry.status !== 'failed') {
      return;
    }

    this.queuedFiles.update((entries) =>
      entries.map((item) =>
        item.id === fileId
          ? {
              ...item,
              status: 'pending',
              uploadProgress: 0,
              uploadedReference: null,
            }
          : item,
      ),
    );
    this.errorMessage.set('');

    if (!this.uploading()) {
      this.uploadPendingFiles();
    }
  }

  cancelFileUpload(fileId: string): void {
    if (this.analyzing()) {
      return;
    }

    const activeUpload = this.activeUploads.get(fileId);
    if (!activeUpload) {
      this.queuedFiles.update((entries) => entries.filter((entry) => entry.id !== fileId));
      this.errorMessage.set('');
      return;
    }

    activeUpload.unsubscribe();
    this.queuedFiles.update((entries) => entries.filter((entry) => entry.id !== fileId));
    this.finishSingleUpload(fileId);
    this.errorMessage.set('');
  }

  private uploadPendingFiles(): void {
    if (this.uploading() || this.analyzing()) {
      return;
    }

    const filesToUpload = this.queuedFiles().filter((entry) => entry.status === 'pending');
    if (filesToUpload.length === 0) {
      return;
    }

    this.queuedFiles.update((entries) =>
      entries.map((entry) => {
        const shouldUpload = filesToUpload.some((candidate) => candidate.id === entry.id);
        if (!shouldUpload) {
          return entry;
        }

        return {
          ...entry,
          status: 'uploading',
          uploadProgress: 0,
        };
      }),
    );

    this.uploading.set(true);
    this.errorMessage.set('');
    this.currentBatch = {
      remaining: filesToUpload.length,
      uploadedCount: 0,
      failedCount: 0,
    };

    for (const entry of filesToUpload) {
      const subscription = this.fileStorage.uploadFile(entry.file, 'invoices').subscribe({
        next: (event) => {
          if (event.type === 'progress') {
            this.updateFileProgress(entry.id, event.progress);
            return;
          }

          this.markFileAsUploaded(entry, event.reference);
        },
        error: () => {
          this.markFileAsFailed(entry.id);
          this.finishSingleUpload(entry.id);
        },
        complete: () => {
          this.finishSingleUpload(entry.id);
        },
      });

      this.activeUploads.set(entry.id, subscription);
    }
  }

  analyzeUploadedFiles(id: string): void {
    if (!this.canAnalyze()) {
      return;
    }

    this.analyzing.set(true);
    this.errorMessage.set('');

    this.invoiceHistoryHttp
      .startAnalysis(id, this.uploadedFiles())
      .pipe(finalize(() => this.analyzing.set(false)))
      .subscribe({
        next: () => {
          this.toast.showSuccess('Listo. Iniciamos el análisis en segundo plano y te avisaremos cuando termine.');
          this.isImportModalOpen.set(false);
          this.resetState();
        },
        error: () => {
          this.errorMessage.set('No fue posible iniciar el análisis del histórico. Intenta nuevamente.');
          this.toast.showError('No fue posible iniciar el análisis en este momento.');
        },
      });
  }

  private resetState(): void {
    this.errorMessage.set('');
    this.queuedFiles.set([]);
  }

  private updateFileProgress(fileId: string, progress: number): void {
    this.queuedFiles.update((entries) =>
      entries.map((entry) => {
        if (entry.id !== fileId || entry.status !== 'uploading') {
          return entry;
        }

        return {
          ...entry,
          uploadProgress: progress,
        };
      }),
    );
  }

  private markFileAsUploaded(entry: InvoiceHistoryQueuedFile, reference: FileReference): void {
    this.queuedFiles.update((entries) =>
      entries.map((queued) => {
        if (queued.id !== entry.id) {
          return queued;
        }

        return {
          ...queued,
          status: 'uploaded',
          uploadProgress: 100,
          uploadedReference: this.toAnalyzeFileReference(entry.file, reference),
        };
      }),
    );

    if (this.currentBatch) {
      this.currentBatch.uploadedCount += 1;
    }
  }

  private markFileAsFailed(fileId: string): void {
    this.queuedFiles.update((entries) =>
      entries.map((entry) => {
        if (entry.id !== fileId) {
          return entry;
        }

        return {
          ...entry,
          status: 'failed',
          uploadProgress: 100,
          uploadedReference: null,
        };
      }),
    );

    if (this.currentBatch) {
      this.currentBatch.failedCount += 1;
    }
  }

  private finishSingleUpload(fileId: string): void {
    this.activeUploads.delete(fileId);

    if (!this.currentBatch) {
      return;
    }

    this.currentBatch.remaining = Math.max(0, this.currentBatch.remaining - 1);
    if (this.currentBatch.remaining > 0) {
      return;
    }

    const { uploadedCount, failedCount } = this.currentBatch;
    this.currentBatch = null;
    this.uploading.set(false);

    if (uploadedCount > 0) {
      this.toast.showSuccess(`Se subieron ${uploadedCount} archivo(s) correctamente.`);
    }
    if (failedCount > 0) {
      this.errorMessage.set('Algunos archivos no se pudieron subir. Elimínalos y vuelve a intentarlo.');
      this.toast.showWarning(`No se pudieron subir ${failedCount} archivo(s).`);
    }

    if (this.hasPendingFiles()) {
      this.uploadPendingFiles();
    }
  }

  private toAnalyzeFileReference(file: File, uploaded: FileReference): InvoiceHistoryAnalyzeFileReference {
    return {
      name: file.name,
      type: file.type || this.detectMimeTypeFromFilename(file.name),
      url: uploaded.key,
    };
  }

  private detectMimeTypeFromFilename(name: string): string {
    const extension = name.split('.').pop()?.toLowerCase() ?? '';

    if (extension === 'xml') {
      return 'application/xml';
    }
    if (extension === 'pdf') {
      return 'application/pdf';
    }
    if (extension === 'zip') {
      return 'application/zip';
    }

    return 'application/octet-stream';
  }
}
