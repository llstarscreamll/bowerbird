import { Component, computed, input, output, signal } from '@angular/core';
import { NgIcon } from '@ng-icons/core';
import { NgScrollbar } from 'ngx-scrollbar';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmEmptyImports } from '@spartan-ng/helm/empty';
import { HlmItemImports } from '@spartan-ng/helm/item';
import { HlmProgressImports } from '@spartan-ng/helm/progress';
import { HlmScrollAreaImports } from '@spartan-ng/helm/scroll-area';
import type { FileUploadQueueItem } from './file-upload.types';

const DEFAULT_MAX_FILE_SIZE_BYTES = 1024 * 1024 * 1024;

@Component({
  selector: 'app-file-upload',
  standalone: true,
  imports: [NgIcon, NgScrollbar, HlmCardImports, HlmButtonImports, HlmBadgeImports, HlmEmptyImports, HlmItemImports, HlmProgressImports, HlmScrollAreaImports, HlmAlertImports],
  template: `
    <div class="space-y-4">
      <input #fileInput type="file" class="hidden" [accept]="accept()" [multiple]="multiple()" [disabled]="isPickerDisabled()" (change)="onFilesSelected($event)" />

      <hlm-card
        class="border-dashed p-6 text-center transition-colors sm:p-8"
        [class.border-primary]="isDragOver()"
        [class.bg-primary/5]="isDragOver()"
        [class.pointer-events-none]="isPickerDisabled()"
        [class.opacity-60]="isPickerDisabled()"
        role="button"
        tabindex="0"
        [attr.aria-label]="dropzoneTitle()"
        [attr.aria-disabled]="isPickerDisabled()"
        [attr.aria-dropeffect]="isDragOver() ? 'copy' : 'none'"
        (click)="openFilePicker(fileInput)"
        (keydown)="onDropzoneKeydown($event, fileInput)"
        (dragenter)="onDragEnter($event)"
        (dragover)="onDragOver($event)"
        (dragleave)="onDragLeave($event)"
        (drop)="onDrop($event)"
      >
        <div class="mx-auto mb-4 flex size-14 items-center justify-center rounded-2xl bg-card text-muted-foreground shadow-sm">
          <ng-icon name="lucideUpload" class="text-3xl" />
        </div>
        <p class="text-base font-semibold">{{ dropzoneTitle() }}</p>
        <p class="mt-1 text-sm text-muted-foreground">{{ dropzoneDescription() }}</p>
        @if (maxFiles()) {
          <p class="mt-1 text-xs text-muted-foreground">Máximo {{ maxFiles() }} archivo(s) · {{ formatBytes(maxFileSizeBytes()) }} por archivo</p>
        }
        <button type="button" hlmBtn variant="outline" class="mt-4" [disabled]="isPickerDisabled()" (click)="openFilePickerFromButton($event, fileInput)">
          <ng-icon name="lucideFolderOpen" />
          {{ browseButtonLabel() }}
        </button>
      </hlm-card>

      @if (validationMessages().length > 0) {
        <hlm-alert variant="destructive">
          <ng-icon hlm name="lucideCircleAlert" />
          <h4 hlmAlertTitle>No se pudieron agregar algunos archivos</h4>
          <div hlmAlertDescription class="space-y-1">
            @for (message of validationMessages(); track message) {
              <p>{{ message }}</p>
            }
          </div>
        </hlm-alert>
      }

      @if (items().length > 0) {
        <div class="space-y-2">
          <div class="flex flex-wrap items-center justify-between gap-2 px-1">
            <div>
              <p class="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{{ listTitle() }}</p>
              <p class="text-xs text-muted-foreground">
                {{ items().length }} archivo(s) · {{ formatBytes(totalBytes()) }}
                @if (uploadSummary()) {
                  · {{ uploadSummary() }}
                }
              </p>
            </div>
            @if (showClearAll()) {
              <button type="button" hlmBtn variant="ghost" size="sm" class="text-muted-foreground" [disabled]="disableActions() && !hasActiveUploads()" (click)="onClearAll()">
                <ng-icon name="lucideTrash2" />
                Limpiar todo
              </button>
            }
          </div>
          <ng-scrollbar hlm class="max-h-48">
            <div hlmItemGroup class="space-y-2 pe-2">
              @for (item of items(); track item.id) {
                <div hlmItem size="sm" variant="outline" class="flex-col items-stretch">
                  <div class="flex w-full flex-wrap items-center gap-3">
                    <div hlmItemMedia>
                      <span hlmBadge [variant]="badgeVariant(item)">{{ fileExtension(item.name) }}</span>
                    </div>
                    <div hlmItemContent class="min-w-0 flex-1">
                      <div hlmItemTitle class="truncate">{{ item.name }}</div>
                      <div hlmItemDescription aria-live="polite" [class.text-destructive]="item.status === 'failed'">{{ formatBytes(item.size) }} · {{ statusLabel(item) }}</div>
                    </div>
                    <div hlmItemActions class="flex gap-1">
                      @if (item.status === 'failed') {
                        <button
                          type="button"
                          hlmBtn
                          variant="ghost"
                          size="icon-sm"
                          [disabled]="disableActions()"
                          (click)="onRetry(item); $event.stopPropagation()"
                          [attr.aria-label]="'Reintentar subida de ' + item.name"
                        >
                          <ng-icon name="lucideRefreshCw" />
                        </button>
                      }
                      <button
                        type="button"
                        hlmBtn
                        variant="ghost"
                        size="icon-sm"
                        [disabled]="disableActions() && item.status !== 'uploading' && item.status !== 'pending'"
                        (click)="onAction(item); $event.stopPropagation()"
                        [attr.aria-label]="actionLabel(item) + ' ' + item.name"
                      >
                        <ng-icon [name]="actionIcon(item)" />
                      </button>
                    </div>
                  </div>
                  @if (item.status !== 'uploaded') {
                    <div hlmItemFooter class="w-full pt-1">
                      <hlm-progress [value]="progressPercent(item)" class="w-full" [class.animate-pulse]="item.status === 'uploading'">
                        <hlm-progress-indicator [class.bg-destructive]="item.status === 'failed'" />
                      </hlm-progress>
                    </div>
                  }
                </div>
              }
            </div>
          </ng-scrollbar>
        </div>
      } @else {
        <hlm-empty class="border border-dashed py-6">
          <ng-icon hlm name="lucideUpload" class="text-muted-foreground/50" />
          <p hlmEmptyTitle class="text-sm font-normal text-muted-foreground">{{ emptyMessage() }}</p>
        </hlm-empty>
      }
    </div>
  `,
})
export class FileUploadComponent {
  readonly accept = input('');
  readonly multiple = input(true);
  readonly items = input<FileUploadQueueItem[]>([]);
  readonly isPickerDisabled = input(false);
  readonly disableActions = input(false);
  readonly validateFile = input<(file: File) => boolean>();
  readonly maxFileSizeBytes = input(DEFAULT_MAX_FILE_SIZE_BYTES);
  readonly maxFiles = input<number | null>(null);
  readonly showClearAll = input(true);
  readonly listTitle = input('Archivos');
  readonly dropzoneTitle = input('Arrastra tus archivos aquí o selecciona');
  readonly dropzoneDescription = input('XML, PDF o ZIP - máximo 1 GB por archivo.');
  readonly browseButtonLabel = input('Buscar archivos');
  readonly emptyMessage = input('Aún no has seleccionado archivos.');

  readonly filesSelected = output<File[]>();
  readonly cancelRequested = output<string>();
  readonly removeRequested = output<string>();
  readonly retryRequested = output<string>();
  readonly clearAllRequested = output<void>();

  readonly validationMessages = signal<string[]>([]);
  readonly isDragOver = signal(false);

  private dragDepth = 0;

  totalBytes(): number {
    return this.items().reduce((sum, item) => sum + item.size, 0);
  }

  uploadSummary(): string | null {
    const queue = this.items();
    if (queue.length === 0) return null;
    const uploaded = queue.filter((item) => item.status === 'uploaded').length;
    const failed = queue.filter((item) => item.status === 'failed').length;
    const inProgress = queue.filter((item) => item.status === 'uploading' || item.status === 'pending').length;

    if (inProgress > 0) {
      return `${uploaded} de ${queue.length} listos`;
    }
    if (failed > 0) {
      return `${failed} con error`;
    }
    if (uploaded === queue.length) {
      return 'Todos listos';
    }
    return null;
  }

  hasActiveUploads(): boolean {
    return this.items().some((item) => item.status === 'uploading' || item.status === 'pending');
  }

  openFilePicker(input: HTMLInputElement): void {
    if (this.isPickerDisabled()) return;
    input.click();
  }

  openFilePickerFromButton(event: MouseEvent, input: HTMLInputElement): void {
    event.stopPropagation();
    this.openFilePicker(input);
  }

  onDropzoneKeydown(event: KeyboardEvent, input: HTMLInputElement): void {
    if (this.isPickerDisabled()) return;
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      this.openFilePicker(input);
    }
  }

  onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const files = input.files ? Array.from(input.files) : [];
    this.emitValidatedFiles(files);
    input.value = '';
  }

  onDragEnter(event: DragEvent): void {
    if (this.isPickerDisabled()) return;
    event.preventDefault();
    this.dragDepth += 1;
    this.isDragOver.set(true);
  }

  onDragOver(event: DragEvent): void {
    if (this.isPickerDisabled()) return;
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'copy';
    }
    this.isDragOver.set(true);
  }

  onDragLeave(event: DragEvent): void {
    event.preventDefault();
    this.dragDepth = Math.max(0, this.dragDepth - 1);
    if (this.dragDepth === 0) {
      this.isDragOver.set(false);
    }
  }

  onDrop(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.dragDepth = 0;
    this.isDragOver.set(false);
    if (this.isPickerDisabled()) return;
    const files = event.dataTransfer?.files ? Array.from(event.dataTransfer.files) : [];
    this.emitValidatedFiles(files);
  }

  onAction(item: FileUploadQueueItem): void {
    if (item.status === 'uploading' || item.status === 'pending') {
      this.cancelRequested.emit(item.id);
      return;
    }
    this.removeRequested.emit(item.id);
  }

  onRetry(item: FileUploadQueueItem): void {
    if (item.status !== 'failed') return;
    this.retryRequested.emit(item.id);
  }

  onClearAll(): void {
    this.clearAllRequested.emit();
  }

  badgeVariant(item: FileUploadQueueItem): 'destructive' | 'secondary' | 'outline' {
    if (item.status === 'failed') return 'destructive';
    if (item.status === 'uploaded') return 'outline';
    return 'secondary';
  }

  statusLabel(item: FileUploadQueueItem): string {
    if (item.status === 'uploaded') return 'Listo para analizar';
    if (item.status === 'failed') return 'Error al subir, intenta de nuevo';
    if (item.status === 'uploading') return `Subiendo (${this.progressPercent(item)}%)`;
    if (item.status === 'pending') return 'En cola';
    return 'Error al subir';
  }

  actionLabel(item: FileUploadQueueItem): string {
    if (item.status === 'uploading' || item.status === 'pending') return 'Cancelar carga de';
    return 'Eliminar';
  }

  actionIcon(item: FileUploadQueueItem): string {
    if (item.status === 'uploading' || item.status === 'pending') return 'lucideX';
    return 'lucideTrash2';
  }

  progressPercent(item: FileUploadQueueItem): number {
    if (item.status === 'uploaded' || item.status === 'failed') return 100;
    if (item.status === 'pending') return 5;
    return Math.min(99, Math.max(5, item.progress));
  }

  fileExtension(name: string): string {
    return name.split('.').pop()?.toUpperCase() ?? 'FILE';
  }

  formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    const units = ['KB', 'MB', 'GB'];
    let size = bytes / 1024;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex += 1;
    }
    return `${size.toFixed(1)} ${units[unitIndex]}`;
  }

  private emitValidatedFiles(files: File[]): void {
    if (files.length === 0) return;

    const messages: string[] = [];
    const accepted: File[] = [];
    const seenInBatch = new Set<string>();
    const queue = this.items();
    const existingFingerprints = new Set(queue.map((item) => this.queueItemFingerprint(item.name, item.size)));
    const maxFiles = this.maxFiles();
    const slotsLeft = maxFiles === null ? Infinity : Math.max(0, maxFiles - queue.length);
    const validateFile = this.validateFile();

    for (const file of files) {
      if (accepted.length >= slotsLeft) {
        messages.push(`Solo puedes agregar hasta ${maxFiles} archivo(s).`);
        break;
      }

      if (validateFile && !validateFile(file)) {
        messages.push(`"${file.name}" no es un tipo de archivo admitido.`);
        continue;
      }

      if (file.size > this.maxFileSizeBytes()) {
        messages.push(`"${file.name}" supera el tamaño máximo de ${this.formatBytes(this.maxFileSizeBytes())}.`);
        continue;
      }

      const batchFingerprint = this.fileFingerprint(file);
      if (seenInBatch.has(batchFingerprint)) {
        messages.push(`"${file.name}" está duplicado en la selección.`);
        continue;
      }

      const queueFingerprint = this.queueItemFingerprint(file.name, file.size);
      if (existingFingerprints.has(queueFingerprint)) {
        messages.push(`"${file.name}" ya está en la lista.`);
        continue;
      }

      seenInBatch.add(batchFingerprint);
      existingFingerprints.add(queueFingerprint);
      accepted.push(file);
    }

    this.validationMessages.set(messages.length > 0 ? [...new Set(messages)] : []);

    if (accepted.length > 0) {
      this.filesSelected.emit(accepted);
    }
  }

  private fileFingerprint(file: File): string {
    return `${file.name}::${file.size}::${file.lastModified}`;
  }

  private queueItemFingerprint(name: string, size: number): string {
    return `${name}::${size}`;
  }
}
