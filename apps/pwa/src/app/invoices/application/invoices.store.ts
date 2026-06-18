import { Injectable, computed, inject, signal } from '@angular/core';
import { ToastService } from '../../core/services/toast.service';
import { InvoicesHttpService } from '../infrastructure/invoices.http.service';
import { InvoiceSummary } from '../domain/invoice.model';

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
}
