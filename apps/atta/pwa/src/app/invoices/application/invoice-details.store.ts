import { Injectable, inject, signal } from '@angular/core';
import { finalize } from 'rxjs/operators';
import { ToastService } from '../../core/services/toast.service';
import { InvoicesHttpService } from '../infrastructure/invoices.http.service';
import { InvoiceDetails } from '../domain/invoice.model';

@Injectable({ providedIn: 'root' })
export class InvoiceDetailsStore {
  private readonly invoicesHttp = inject(InvoicesHttpService);
  private readonly toast = inject(ToastService);

  readonly invoice = signal<InvoiceDetails | null>(null);
  readonly isLoading = signal(false);
  readonly isDownloading = signal(false);

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

  downloadOriginalDocument(): void {
    const invoice = this.invoice();
    if (!invoice || this.isDownloading()) {
      return;
    }

    this.isDownloading.set(true);
    this.invoicesHttp
      .downloadOriginalDocument(invoice.id)
      .pipe(finalize(() => this.isDownloading.set(false)))
      .subscribe({
        next: (response) => {
          const filename = filenameFromContentDisposition(response.headers.get('Content-Disposition'), fallbackInvoiceFilename(invoice));
          const blob = response.body ?? new Blob();
          const url = URL.createObjectURL(blob);
          const link = document.createElement('a');
          link.href = url;
          link.download = filename;
          link.click();
          URL.revokeObjectURL(url);
        },
        error: () => {
          this.toast.showError('No se pudo descargar el archivo original de la factura.');
        },
      });
  }
}

function fallbackInvoiceFilename(invoice: InvoiceDetails): string {
  const number = invoice.invoice_number?.trim();
  return number ? `factura-${number}` : 'factura';
}

function filenameFromContentDisposition(header: string | null, fallback: string): string {
  if (!header) {
    return fallback;
  }
  const quoted = /filename="([^"]+)"/i.exec(header);
  if (quoted?.[1]) {
    return quoted[1];
  }
  const plain = /filename=([^;]+)/i.exec(header);
  if (plain?.[1]) {
    return plain[1].trim();
  }
  return fallback;
}
