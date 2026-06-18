import { Injectable, inject, signal } from '@angular/core';
import { ToastService } from '../../core/services/toast.service';
import { InvoicesHttpService } from '../infrastructure/invoices.http.service';
import { InvoiceDetails } from '../domain/invoice.model';

@Injectable({ providedIn: 'root' })
export class InvoiceDetailsStore {
  private readonly invoicesHttp = inject(InvoicesHttpService);
  private readonly toast = inject(ToastService);

  readonly invoice = signal<InvoiceDetails | null>(null);
  readonly isLoading = signal(false);

  loadInvoice(id: string): void {
    this.isLoading.set(true);
    this.invoice.set(null);
    this.invoicesHttp.getInvoiceById(id).subscribe({
      next: (details) => {
        this.invoice.set(details);
        this.isLoading.set(false);
      },
      error: () => {
        this.toast.showError('No se pudo cargar los detalles de la factura.');
        this.isLoading.set(false);
      },
    });
  }
}
