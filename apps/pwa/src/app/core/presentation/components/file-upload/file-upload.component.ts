import { CommonModule } from '@angular/common';
import { Component, EventEmitter, Input, Output } from '@angular/core';

export type FileUploadQueueItemStatus = 'pending' | 'uploading' | 'uploaded' | 'failed';

export interface FileUploadQueueItem {
  id: string;
  name: string;
  size: number;
  status: FileUploadQueueItemStatus;
  progress: number;
}

@Component({
  selector: 'app-file-upload',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="space-y-4">
      <input #fileInput type="file" class="hidden" [accept]="accept" [multiple]="multiple" [disabled]="isPickerDisabled" (change)="onFilesSelected($event)" />

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
        <p class="text-base font-semibold text-slate-900 dark:text-slate-100">{{ dropzoneTitle }}</p>
        <p class="mt-1 text-sm text-slate-500 dark:text-slate-300">{{ dropzoneDescription }}</p>
        <button type="button" class="btn-secondary mt-4" [disabled]="isPickerDisabled" (click)="openFilePicker(fileInput)">
          <span class="material-icons-outlined text-[18px] mr-1.5">folder_open</span>
          {{ browseButtonLabel }}
        </button>
      </div>

      <div *ngIf="items.length > 0" class="space-y-2">
        <div class="flex items-center justify-between px-1">
          <p class="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-300">{{ listTitle }}</p>
          <p class="text-xs text-slate-500 dark:text-slate-300">{{ items.length }} archivo(s)</p>
        </div>
        <div class="space-y-2 max-h-48 overflow-y-auto pr-1">
          <div *ngFor="let item of items; trackBy: trackByFileId" class="rounded-xl border border-slate-200 bg-white px-3 pt-5 pb-2 dark:border-slate-700 dark:bg-slate-900/70">
            <div class="flex items-center gap-3">
              <div
                class="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:bg-slate-800 dark:text-slate-200"
              >
                {{ fileExtension(item.name) }}
              </div>
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm font-semibold text-slate-900 dark:text-slate-100">{{ item.name }}</p>
                <p class="text-sm text-slate-500 dark:text-slate-300">{{ formatBytes(item.size) }} - {{ statusLabel(item) }}</p>
              </div>
              <button
                type="button"
                class="rounded-lg p-1.5 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-800 disabled:cursor-not-allowed disabled:opacity-40 dark:text-slate-300 dark:hover:bg-slate-800 dark:hover:text-slate-100"
                [disabled]="disableActions"
                (click)="onAction(item)"
                [attr.aria-label]="actionLabel(item) + ' ' + item.name"
              >
                <span class="material-icons-outlined text-[18px]">{{ actionIcon(item) }}</span>
              </button>
            </div>

            <div class="mt-2 h-1 w-full overflow-hidden rounded-full" [class.bg-slate-200]="item.status !== 'uploaded'" [class.dark:bg-slate-700]="item.status !== 'uploaded'">
              <div
                class="h-full rounded-full"
                [style.width.%]="progressPercent(item)"
                [class.opacity-0]="item.status === 'uploaded'"
                [class.bg-rose-500]="item.status === 'failed'"
                [class.animate-pulse]="item.status === 'uploading'"
                [class.bg-indigo-600]="item.status !== 'failed'"
              ></div>
            </div>
          </div>
        </div>
      </div>

      <div *ngIf="items.length === 0" class="rounded-xl border border-slate-200 bg-slate-50 p-3 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/50 dark:text-slate-300">
        {{ emptyMessage }}
      </div>
    </div>
  `,
})
export class FileUploadComponent {
  @Input() accept = '';
  @Input() multiple = true;
  @Input() items: FileUploadQueueItem[] = [];
  @Input() isPickerDisabled = false;
  @Input() disableActions = false;
  @Input() listTitle = 'Archivos';
  @Input() dropzoneTitle = 'Arrastra tus archivos aquí o selecciona';
  @Input() dropzoneDescription = 'XML, PDF o ZIP - máximo 1 GB por archivo.';
  @Input() browseButtonLabel = 'Buscar archivos';
  @Input() emptyMessage = 'Aun no has seleccionado archivos.';

  @Output() filesSelected = new EventEmitter<File[]>();
  @Output() cancelRequested = new EventEmitter<string>();
  @Output() removeRequested = new EventEmitter<string>();

  isDragOver = false;

  openFilePicker(input: HTMLInputElement): void {
    if (this.isPickerDisabled) {
      return;
    }

    input.click();
  }

  onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const files = input.files ? Array.from(input.files) : [];

    this.filesSelected.emit(files);
    input.value = '';
  }

  onDragOver(event: DragEvent): void {
    if (this.isPickerDisabled) {
      return;
    }

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

    if (this.isPickerDisabled) {
      return;
    }

    const files = event.dataTransfer?.files ? Array.from(event.dataTransfer.files) : [];
    this.filesSelected.emit(files);
  }

  onAction(item: FileUploadQueueItem): void {
    if (item.status === 'uploading' || item.status === 'pending') {
      this.cancelRequested.emit(item.id);
      return;
    }

    this.removeRequested.emit(item.id);
  }

  statusLabel(item: FileUploadQueueItem): string {
    if (item.status === 'uploaded') {
      return 'Listo para analizar';
    }
    if (item.status === 'failed') {
      return 'Error al subir, intenta de nuevo';
    }
    if (item.status === 'uploading') {
      return `Subiendo archivo (${this.progressPercent(item)}%)`;
    }
    if (item.status === 'pending') {
      return 'En cola';
    }

    return 'Error al subir';
  }

  actionLabel(item: FileUploadQueueItem): string {
    if (item.status === 'uploading' || item.status === 'pending') {
      return 'Cancelar carga de';
    }

    return 'Eliminar';
  }

  actionIcon(item: FileUploadQueueItem): string {
    if (item.status === 'uploading' || item.status === 'pending') {
      return 'close';
    }

    return 'delete';
  }

  progressPercent(item: FileUploadQueueItem): number {
    if (item.status === 'uploaded' || item.status === 'failed') {
      return 100;
    }

    if (item.status === 'pending') {
      return 5;
    }

    return Math.min(99, Math.max(5, item.progress));
  }

  fileExtension(name: string): string {
    return name.split('.').pop()?.toUpperCase() ?? 'FILE';
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

  trackByFileId(_: number, item: FileUploadQueueItem): string {
    return item.id;
  }
}
