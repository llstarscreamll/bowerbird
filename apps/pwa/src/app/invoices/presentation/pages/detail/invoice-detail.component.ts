import { CommonModule, CurrencyPipe, DatePipe } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { InvoiceDetailsStore } from '../../../application/invoice-details.store';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSeparatorImports } from '@spartan-ng/helm/separator';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { HlmTableImports } from '@spartan-ng/helm/table';

@Component({
  selector: 'app-invoice-detail',
  standalone: true,
  imports: [CommonModule, CurrencyPipe, DatePipe, RouterLink, NgIcon, HlmCardImports, HlmButtonImports, HlmSpinnerImports, HlmTableImports, HlmSeparatorImports],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto bg-muted/20 p-4 sm:p-8' },
  template: `
    <div class="mx-auto w-full max-w-5xl space-y-6">
      @if (isLoading()) {
        <div class="flex items-center justify-center py-20">
          <hlm-spinner class="size-8 text-primary" />
        </div>
      }

      @if (invoice(); as inv) {
        <a routerLink="../" class="inline-flex items-center text-sm font-medium text-muted-foreground hover:text-foreground">
          <ng-icon name="lucideArrowLeft" class="mr-1" />
          Volver a facturas
        </a>

        <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
          <div>
            <h2 class="text-2xl font-bold tracking-tight sm:text-3xl">Factura {{ inv.invoice_number || 'N/A' }}</h2>
            <p class="mt-1 text-sm text-muted-foreground">Emitida el {{ inv.issue_date | date: 'longDate' }}</p>
          </div>
          <button hlmBtn variant="outline">
            <ng-icon name="lucideDownload" />
            Descargar
          </button>
        </div>

        <div class="mt-6 grid grid-cols-1 gap-6 md:grid-cols-2">
          <hlm-card class="flex flex-col justify-between p-6">
            <div>
              <h3 class="mb-4 text-sm font-medium uppercase tracking-wider text-muted-foreground">Emisor</h3>
              <p class="mb-1 text-lg font-semibold">{{ inv.issuer_name || 'Desconocido' }}</p>
              <p class="text-sm text-muted-foreground">
                NIT: <span class="font-medium text-foreground">{{ inv.issuer_tax_id || 'N/A' }}</span>
              </p>
            </div>
            <div class="mt-4 border-t pt-4">
              <p class="text-sm text-muted-foreground">CUFE</p>
              <p class="mt-1 break-all rounded bg-muted p-2 font-mono text-xs">{{ inv.cufe || 'N/A' }}</p>
            </div>
          </hlm-card>

          <hlm-card class="flex flex-col justify-between p-6">
            <div>
              <h3 class="mb-4 text-sm font-medium uppercase tracking-wider text-muted-foreground">Receptor</h3>
              <p class="mb-1 text-lg font-semibold">{{ inv.receiver_name || 'Desconocido' }}</p>
              <p class="text-sm text-muted-foreground">
                NIT: <span class="font-medium text-foreground">{{ inv.receiver_tax_id || 'N/A' }}</span>
              </p>
            </div>
            <div class="mt-4 grid grid-cols-2 gap-4 border-t pt-4">
              <div>
                <p class="text-sm text-muted-foreground">Vencimiento</p>
                <p class="mt-1 text-sm font-medium">{{ inv.due_date ? (inv.due_date | date: 'mediumDate') : 'N/A' }}</p>
              </div>
              <div>
                <p class="text-sm text-muted-foreground">Método de pago</p>
                <p class="mt-1 text-sm font-medium">{{ inv.payment_code || 'N/A' }}</p>
              </div>
            </div>
          </hlm-card>
        </div>

        <hlm-card class="mt-6 overflow-hidden p-0">
          <div class="border-b px-6 py-4">
            <h3 class="text-lg font-medium">Detalle de Productos o Servicios</h3>
          </div>
          <div class="overflow-x-auto">
            <table hlmTable>
              <thead hlmTHead>
                <tr hlmTr>
                  <th hlmTh>Código</th>
                  <th hlmTh>Descripción</th>
                  <th hlmTh class="text-right">Cantidad</th>
                  <th hlmTh class="text-right">Precio Unitario</th>
                  <th hlmTh class="text-right">Impuesto</th>
                  <th hlmTh class="text-right">Total</th>
                </tr>
              </thead>
              <tbody hlmTBody>
                @for (line of inv.lines; track $index) {
                  <tr hlmTr>
                    <td hlmTd class="text-muted-foreground">{{ line.item_code || 'N/A' }}</td>
                    <td hlmTd>
                      <div class="max-w-[300px] break-words font-medium">{{ line.description }}</div>
                    </td>
                    <td hlmTd class="text-right text-muted-foreground">{{ line.quantity }}</td>
                    <td hlmTd class="text-right text-muted-foreground">{{ line.unit_price | currency: inv.currency_code : 'symbol' : '1.2-2' }}</td>
                    <td hlmTd class="text-right text-muted-foreground">{{ line.line_tax_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}</td>
                    <td hlmTd class="text-right font-medium">{{ line.line_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}</td>
                  </tr>
                }
                @if (!inv.lines || inv.lines.length === 0) {
                  <tr hlmTr>
                    <td hlmTd colspan="6" class="py-8 text-center text-muted-foreground">No se encontraron líneas de detalle para esta factura.</td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
        </hlm-card>

        <div class="mt-6 flex justify-end">
          <hlm-card class="w-full overflow-hidden p-0 sm:w-80">
            <div class="border-b bg-muted/30 px-6 py-4">
              <h3 class="text-sm font-medium">Resumen de Totales</h3>
            </div>
            <div class="space-y-3 px-6 py-4">
              <div class="flex justify-between text-sm">
                <span class="text-muted-foreground">Subtotal</span>
                <span class="font-medium">{{ inv.subtotal | currency: inv.currency_code : 'symbol' : '1.2-2' }}</span>
              </div>
              <div class="flex justify-between text-sm">
                <span class="text-muted-foreground">Impuestos</span>
                <span class="font-medium">{{ inv.tax_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}</span>
              </div>
              <hlm-separator />
              <div class="flex justify-between items-center">
                <span class="text-base font-semibold">Total a Pagar</span>
                <span class="text-lg font-bold text-primary">{{ inv.grand_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}</span>
              </div>
            </div>
          </hlm-card>
        </div>
      }
    </div>
  `,
})
export class InvoiceDetailComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly store = inject(InvoiceDetailsStore);

  readonly invoice = this.store.invoice;
  readonly isLoading = this.store.isLoading;

  ngOnInit(): void {
    const invoiceId = this.route.snapshot.paramMap.get('invoiceId');
    if (invoiceId) {
      this.store.loadInvoice(invoiceId);
    }
  }
}
