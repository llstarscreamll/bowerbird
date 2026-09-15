import { Injectable, computed, inject, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable, Subscription, catchError, of, tap } from 'rxjs';
import { FileStorageService } from '../../core/services/file-storage.service';
import { ToastService } from '../../core/services/toast.service';
import { FileUploadQueueItem } from '../../core/presentation/components/file-upload';
import { CatalogHttpService } from '../infrastructure/catalog.http.service';
import { CATALOG_IMPORT_MAX_FILE_BYTES, CATALOG_IMPORT_UPLOAD_MODULE, CatalogImport, CatalogImportError, catalogImportIsActive } from '../domain/catalog-import.model';
import { generateUlid } from '../../core/utils/ulid';

@Injectable({ providedIn: 'root' })
export class CatalogImportStore {
  private readonly http = inject(CatalogHttpService);
  private readonly files = inject(FileStorageService);
  private readonly toast = inject(ToastService);

  readonly imports = signal<CatalogImport[]>([]);
  readonly selected = signal<CatalogImport | null>(null);
  readonly errors = signal<CatalogImportError[]>([]);
  readonly errorsTotal = signal(0);
  readonly loading = signal(false);
  readonly loadingMore = signal(false);
  readonly loadingErrors = signal(false);
  readonly loadingMoreErrors = signal(false);
  readonly submitting = signal(false);
  readonly errorMessage = signal<string | null>(null);
  readonly hasMore = signal(false);
  readonly errorsHasMore = signal(false);
  readonly dialogOpen = signal(false);
  readonly uploadItem = signal<FileUploadQueueItem | null>(null);
  readonly fileKey = signal<string | null>(null);
  readonly cancelling = signal(false);

  private listCursor = signal<string | undefined>(undefined);
  private errorCursor = signal<string | undefined>(undefined);
  private uploadSub?: Subscription;

  readonly activeImport = computed(() => {
    const selected = this.selected();
    if (selected && catalogImportIsActive(selected.status)) return selected;
    return this.imports().find((item) => catalogImportIsActive(item.status)) ?? null;
  });

  readonly uploadQueue = computed<FileUploadQueueItem[]>(() => {
    const item = this.uploadItem();
    return item ? [item] : [];
  });

  readonly canSubmitImport = computed(() => Boolean(this.fileKey()) && !this.submitting() && this.uploadItem()?.status === 'uploaded');

  loadImports(): void {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.listCursor.set(undefined);
    this.http.listImports(20).subscribe({
      next: (page) => {
        this.imports.set(page.items);
        this.hasMore.set(page.hasMore);
        this.listCursor.set(page.cursor);
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => this.handleError(err, 'No se pudo cargar el historial de importaciones.'),
    });
  }

  loadMoreImports(): void {
    if (!this.hasMore() || this.loadingMore()) return;
    this.loadingMore.set(true);
    this.http.listImports(20, this.listCursor()).subscribe({
      next: (page) => {
        this.imports.update((current) => [...current, ...page.items]);
        this.hasMore.set(page.hasMore);
        this.listCursor.set(page.cursor);
        this.loadingMore.set(false);
      },
      error: (err: HttpErrorResponse) => {
        this.loadingMore.set(false);
        this.handleError(err, 'No se pudieron cargar más importaciones.');
      },
    });
  }

  loadImport(id: string): void {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.http.getImport(id).subscribe({
      next: (imp) => {
        this.selected.set(imp);
        this.loading.set(false);
        this.loadErrors(id, true);
      },
      error: (err: HttpErrorResponse) => this.handleError(err, 'No se pudo cargar la importación.'),
    });
  }

  refreshImport(id: string): void {
    this.http.getImport(id).subscribe({
      next: (imp) => {
        const prevFailed = this.selected()?.failed_count ?? 0;
        this.selected.set(imp);
        if (imp.failed_count !== prevFailed) {
          this.loadErrors(id, true);
        }
      },
    });
  }

  loadErrors(importId: string, reset = false): void {
    if (reset) {
      this.errorCursor.set(undefined);
      this.errors.set([]);
    }
    this.loadingErrors.set(true);
    this.http.listImportErrors(importId, 50, reset ? undefined : this.errorCursor()).subscribe({
      next: (page) => {
        this.errors.set(reset ? page.items : [...this.errors(), ...page.items]);
        this.errorsHasMore.set(page.hasMore);
        this.errorCursor.set(page.cursor);
        this.errorsTotal.set(page.total ?? page.items.length);
        this.loadingErrors.set(false);
      },
      error: () => {
        this.loadingErrors.set(false);
      },
    });
  }

  loadMoreErrors(importId: string): void {
    if (!this.errorsHasMore() || this.loadingMoreErrors()) return;
    this.loadingMoreErrors.set(true);
    this.http.listImportErrors(importId, 50, this.errorCursor()).subscribe({
      next: (page) => {
        this.errors.update((current) => [...current, ...page.items]);
        this.errorsHasMore.set(page.hasMore);
        this.errorCursor.set(page.cursor);
        if (page.total != null) this.errorsTotal.set(page.total);
        this.loadingMoreErrors.set(false);
      },
      error: () => {
        this.loadingMoreErrors.set(false);
      },
    });
  }

  cancel(id: string): Observable<CatalogImport | null> {
    this.cancelling.set(true);
    this.errorMessage.set(null);
    return this.http.cancelImport(id).pipe(
      tap((imp) => {
        this.cancelling.set(false);
        this.selected.set(imp);
        this.toast.showSuccess('Importación cancelada.');
      }),
      catchError((err: HttpErrorResponse) => {
        this.cancelling.set(false);
        this.handleError(err, 'No se pudo cancelar la importación.');
        return of(null);
      }),
    );
  }

  openDialog(): void {
    this.dialogOpen.set(true);
    this.errorMessage.set(null);
    this.resetUpload();
  }

  closeDialog(): void {
    if (this.submitting()) return;
    this.dialogOpen.set(false);
    this.resetUpload();
  }

  addFile(file: File): void {
    this.resetUpload();
    this.uploadItem.set({ id: '1', name: file.name, size: file.size, status: 'uploading', progress: 0 });
    this.uploadSub = this.files.uploadFile(file, CATALOG_IMPORT_UPLOAD_MODULE).subscribe({
      next: (event) => {
        if (event.type === 'progress') {
          this.uploadItem.update((item) => (item ? { ...item, progress: event.progress } : item));
        }
        if (event.type === 'completed') {
          this.fileKey.set(event.reference.key);
          this.uploadItem.update((item) => (item ? { ...item, status: 'uploaded', progress: 100 } : item));
        }
      },
      error: () => {
        this.uploadItem.update((item) => (item ? { ...item, status: 'failed' } : item));
        this.toast.showError('No se pudo subir el archivo CSV.');
      },
    });
  }

  removeFile(): void {
    this.uploadSub?.unsubscribe();
    this.resetUpload();
  }

  queueImport(): Observable<CatalogImport | null> {
    const key = this.fileKey();
    if (!key) return of(null);
    this.submitting.set(true);
    this.errorMessage.set(null);
    return this.http.queueImport(generateUlid(), key).pipe(
      tap((imp) => {
        this.submitting.set(false);
        this.dialogOpen.set(false);
        this.resetUpload();
        this.toast.showSuccess('Importación encolada.');
      }),
      catchError((err: HttpErrorResponse) => {
        this.submitting.set(false);
        this.handleError(err, 'No se pudo encolar la importación.');
        return of(null);
      }),
    );
  }

  downloadTemplate(): void {
    this.http.downloadImportTemplate().subscribe({
      next: (blob) => {
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'catalog-import-template.csv';
        a.click();
        URL.revokeObjectURL(url);
      },
      error: () => this.toast.showError('No se pudo descargar la plantilla.'),
    });
  }

  validateCsv(file: File): boolean {
    const name = file.name.toLowerCase();
    return name.endsWith('.csv') && file.size <= CATALOG_IMPORT_MAX_FILE_BYTES;
  }

  private resetUpload(): void {
    this.uploadSub?.unsubscribe();
    this.uploadSub = undefined;
    this.uploadItem.set(null);
    this.fileKey.set(null);
  }

  private handleError(err: HttpErrorResponse, fallback: string): void {
    this.loading.set(false);
    this.submitting.set(false);
    if (err.status >= 400 && err.status < 500) {
      this.errorMessage.set(err.error?.errors?.[0]?.detail || fallback);
    } else {
      this.toast.showError(fallback);
    }
  }
}
