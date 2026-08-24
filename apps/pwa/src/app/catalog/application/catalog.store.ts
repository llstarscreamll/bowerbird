import { Injectable, inject, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { ToastService } from '../../core/services/toast.service';
import { CatalogHttpService } from '../infrastructure/catalog.http.service';
import { CatalogItem, CatalogReviewLine, LineDecisionPayload } from '../domain/catalog.model';

@Injectable({ providedIn: 'root' })
export class CatalogStore {
  private readonly http = inject(CatalogHttpService);
  private readonly toast = inject(ToastService);

  readonly items = signal<CatalogItem[]>([]);
  readonly reviewLines = signal<CatalogReviewLine[]>([]);
  readonly loading = signal(false);
  readonly errorMessage = signal<string | null>(null);

  loadItems(status?: string): void {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.http.listItems(undefined, status).subscribe({
      next: (items) => {
        this.items.set(items);
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => this.handleError(err, 'No se pudieron cargar los ítems.'),
    });
  }

  loadReviewQueue(): void {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.http.listReviewQueue().subscribe({
      next: (lines) => {
        this.reviewLines.set(lines);
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => this.handleError(err, 'No se pudo cargar la cola de revisión.'),
    });
  }

  rememberDecision(lineId: string, payload: LineDecisionPayload, onDone?: () => void): void {
    this.errorMessage.set(null);
    this.http.rememberDecision(lineId, payload).subscribe({
      next: () => {
        this.toast.showSuccess('Decisión de coincidencia guardada.');
        onDone?.();
      },
      error: (err: HttpErrorResponse) => this.handleError(err, 'No se pudo guardar la decisión.'),
    });
  }

  private handleError(err: HttpErrorResponse, fallback: string): void {
    this.loading.set(false);
    if (err.status >= 400 && err.status < 500) {
      this.errorMessage.set(err.error?.errors?.[0]?.detail || fallback);
    } else {
      this.toast.showError(fallback);
    }
  }
}
