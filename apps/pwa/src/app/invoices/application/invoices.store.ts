import { Injectable, computed, inject, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable, catchError, map, of, tap } from 'rxjs';
import { ToastService } from '../../core/services/toast.service';
import { InvoicesHttpService } from '../infrastructure/invoices.http.service';
import { InvoiceSummary, InvoiceReviewLine, LineDecisionPayload } from '../domain/invoice.model';

@Injectable({ providedIn: 'root' })
export class InvoicesStore {
  private readonly invoicesHttp = inject(InvoicesHttpService);
  private readonly toast = inject(ToastService);

  readonly invoices = signal<InvoiceSummary[]>([]);
  readonly isLoading = signal(false);
  readonly isLoadingMore = signal(false);
  readonly limit = signal(20);
  readonly cursor = signal<string | undefined>(undefined);
  readonly hasMore = signal(false);

  readonly hasInvoices = computed(() => this.invoices().length > 0);

  readonly reviewLines = signal<InvoiceReviewLine[]>([]);
  readonly reviewLoading = signal(false);
  readonly reviewErrorMessage = signal<string | null>(null);

  loadInvoices(): void {
    this.isLoading.set(true);
    this.invoicesHttp.listInvoices(this.limit()).subscribe({
      next: (response) => {
        this.invoices.set(response.items);
        this.hasMore.set(response.has_more);
        this.cursor.set(response.cursor);
        this.isLoading.set(false);
      },
      error: () => {
        this.toast.showError('No se pudieron cargar las facturas en este momento.');
        this.isLoading.set(false);
      },
    });
  }

  loadMore(): void {
    if (!this.hasMore() || this.isLoadingMore()) {
      return;
    }

    this.isLoadingMore.set(true);
    this.invoicesHttp.listInvoices(this.limit(), this.cursor()).subscribe({
      next: (response) => {
        this.invoices.update((current) => [...current, ...response.items]);
        this.hasMore.set(response.has_more);
        this.cursor.set(response.cursor);
        this.isLoadingMore.set(false);
      },
      error: () => {
        this.toast.showError('No se pudieron cargar más facturas.');
        this.isLoadingMore.set(false);
      },
    });
  }

  loadReviewQueue(): void {
    this.reviewLoading.set(true);
    this.reviewErrorMessage.set(null);
    this.invoicesHttp.listReviewQueue().subscribe({
      next: (lines) => {
        this.reviewLines.set(lines);
        this.reviewLoading.set(false);
      },
      error: (err: HttpErrorResponse) => this.handleReviewError(err, 'No se pudo cargar la cola de revisión.'),
    });
  }

  resolveLineDecision(invoiceId: string, lineId: string, payload: LineDecisionPayload): Observable<boolean> {
    this.reviewErrorMessage.set(null);
    return this.invoicesHttp.applyLineDecision(invoiceId, lineId, payload).pipe(
      tap(() => this.toast.showSuccess('Decisión de coincidencia guardada.')),
      map(() => true),
      catchError((err: HttpErrorResponse) => {
        this.handleReviewError(err, 'No se pudo guardar la decisión.');
        return of(false);
      }),
    );
  }

  private handleReviewError(err: HttpErrorResponse, fallback: string): void {
    this.reviewLoading.set(false);
    if (err.status >= 400 && err.status < 500) {
      this.reviewErrorMessage.set(err.error?.errors?.[0]?.detail || fallback);
    } else {
      this.toast.showError(fallback);
    }
  }
}
