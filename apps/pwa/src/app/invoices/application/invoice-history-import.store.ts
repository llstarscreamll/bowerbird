import { Injectable, computed, inject, signal } from '@angular/core';
import { catchError, finalize, forkJoin, map, of } from 'rxjs';
import { FileReference } from '../../core/domain/file-storage.model';
import { FileStorageService } from '../../core/services/file-storage.service';
import { ToastService } from '../../core/services/toast.service';
import { InvoiceHistoryAnalyzeFileReference } from '../domain/invoice-history-analysis.model';
import { supportsInvoiceHistoryFile } from '../domain/invoice-history-import.model';
import { InvoiceHistoryHttpService } from '../infrastructure/invoice-history.http.service';

type InvoiceHistoryQueueStatus = 'pending' | 'uploading' | 'uploaded' | 'failed';

export interface InvoiceHistoryQueuedFile {
  id: string;
  file: File;
  status: InvoiceHistoryQueueStatus;
  uploadedReference: InvoiceHistoryAnalyzeFileReference | null;
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

    const existingFingerprints = new Set(this.queuedFiles().map((entry) => this.fileFingerprint(entry.file)));

    this.errorMessage.set('');

    const newEntries: InvoiceHistoryQueuedFile[] = [];
    const invalidFiles: string[] = [];
    const duplicatedFiles: string[] = [];

    for (const file of files) {
      if (!supportsInvoiceHistoryFile(file)) {
        invalidFiles.push(file.name);
        continue;
      }

      const fingerprint = this.fileFingerprint(file);
      if (existingFingerprints.has(fingerprint)) {
        duplicatedFiles.push(file.name);
        continue;
      }

      existingFingerprints.add(fingerprint);
      newEntries.push({
        id: String(this.nextId++),
        file,
        status: 'pending',
        uploadedReference: null,
      });
    }

    if (invalidFiles.length > 0) {
      this.toast.showWarning(`Se omitieron archivos no soportados: ${invalidFiles.join(', ')}.`);
    }

    if (duplicatedFiles.length > 0) {
      this.toast.showWarning(`Se omitieron archivos duplicados: ${duplicatedFiles.join(', ')}.`);
    }

    if (newEntries.length > 0) {
      this.queuedFiles.update((entries) => [...entries, ...newEntries]);
    }

    if (this.queuedFiles().length === 0) {
      this.errorMessage.set('Selecciona archivos XML, PDF o ZIP para importar el historico.');
      return;
    }

    if (!this.uploading()) {
      this.uploadPendingFiles();
    }
  }

  removeFile(fileId: string): void {
    if (this.uploading() || this.analyzing()) {
      return;
    }

    this.queuedFiles.update((entries) => entries.filter((entry) => entry.id !== fileId));
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
        };
      }),
    );

    this.uploading.set(true);
    this.errorMessage.set('');

    forkJoin(
      filesToUpload.map((entry) =>
        this.fileStorage.uploadFile(entry.file, 'invoices').pipe(
          map((reference) => ({ id: entry.id, reference, ok: true as const })),
          catchError(() => of({ id: entry.id, reference: null, ok: false as const })),
        ),
      ),
    )
      .pipe(finalize(() => this.uploading.set(false)))
      .subscribe({
        next: (results) => {
          const resultById = new Map(results.map((result) => [result.id, result]));

          this.queuedFiles.update((entries) =>
            entries.map((entry) => {
              const result = resultById.get(entry.id);
              if (!result) {
                return entry;
              }

              if (!result.ok || !result.reference) {
                return {
                  ...entry,
                  status: 'failed',
                  uploadedReference: null,
                };
              }

              return {
                ...entry,
                status: 'uploaded',
                uploadedReference: this.toAnalyzeFileReference(entry.file, result.reference),
              };
            }),
          );

          const uploadedCount = results.filter((result) => result.ok).length;
          const failedCount = results.length - uploadedCount;

          if (uploadedCount > 0) {
            this.toast.showSuccess(`Se subieron ${uploadedCount} archivo(s) correctamente.`);
          }
          if (failedCount > 0) {
            this.errorMessage.set('Algunos archivos no se pudieron subir. Eliminalos y vuelve a intentarlo.');
            this.toast.showWarning(`No se pudieron subir ${failedCount} archivo(s).`);
          }

          if (this.hasPendingFiles()) {
            this.uploadPendingFiles();
          }
        },
      });
  }

  analyzeUploadedFiles(): void {
    if (!this.canAnalyze()) {
      return;
    }

    this.analyzing.set(true);
    this.errorMessage.set('');

    this.invoiceHistoryHttp
      .startAnalysis(this.uploadedFiles())
      .pipe(finalize(() => this.analyzing.set(false)))
      .subscribe({
        next: () => {
          this.toast.showSuccess('Listo. Iniciamos el analisis en segundo plano y te avisaremos cuando termine.');
          this.isImportModalOpen.set(false);
          this.resetState();
        },
        error: () => {
          this.errorMessage.set('No fue posible iniciar el analisis del historico. Intenta nuevamente.');
          this.toast.showError('No fue posible iniciar el analisis en este momento.');
        },
      });
  }

  private resetState(): void {
    this.errorMessage.set('');
    this.queuedFiles.set([]);
  }

  private fileFingerprint(file: File): string {
    return `${file.name}::${file.size}::${file.lastModified}`;
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
