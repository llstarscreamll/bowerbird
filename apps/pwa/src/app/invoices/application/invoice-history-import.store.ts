import { Injectable, inject, signal } from '@angular/core';
import { finalize, forkJoin } from 'rxjs';
import { FileReference } from '../../core/domain/file-storage.model';
import { FileStorageService } from '../../core/services/file-storage.service';
import { ToastService } from '../../core/services/toast.service';
import { supportsInvoiceHistoryFile } from '../domain/invoice-history-import.model';

@Injectable({ providedIn: 'root' })
export class InvoiceHistoryImportStore {
  readonly uploading = signal(false);
  readonly errorMessage = signal('');
  readonly uploadedFiles = signal<FileReference[]>([]);

  private readonly fileStorage = inject(FileStorageService);
  private readonly toast = inject(ToastService);

  importFiles(files: readonly File[]): void {
    if (files.length === 0 || this.uploading()) {
      return;
    }

    this.errorMessage.set('');

    const validFiles: File[] = [];
    const invalidFiles: string[] = [];

    for (const file of files) {
      if (supportsInvoiceHistoryFile(file)) {
        validFiles.push(file);
      } else {
        invalidFiles.push(file.name);
      }
    }

    if (invalidFiles.length > 0) {
      this.toast.showWarning(`Se omitieron archivos no soportados: ${invalidFiles.join(', ')}.`);
    }

    if (validFiles.length === 0) {
      this.errorMessage.set('Selecciona archivos XML, PDF o ZIP para importar el historico.');
      return;
    }

    this.uploading.set(true);

    forkJoin(validFiles.map((file) => this.fileStorage.uploadFile(file, 'invoices')))
      .pipe(finalize(() => this.uploading.set(false)))
      .subscribe({
        next: (references) => {
          this.uploadedFiles.set(references);
          this.toast.showSuccess(`Se importaron ${references.length} archivo(s) correctamente.`);
        },
        error: () => {
          this.errorMessage.set('No fue posible importar el historico de facturas.');
          this.toast.showError('No fue posible importar los archivos seleccionados.');
        },
      });
  }
}
