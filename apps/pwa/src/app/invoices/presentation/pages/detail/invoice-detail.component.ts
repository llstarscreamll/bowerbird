import { CommonModule, CurrencyPipe, DatePipe } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { InvoiceDetailsStore } from '../../../application/invoice-details.store';

@Component({
  selector: 'app-invoice-detail',
  standalone: true,
  imports: [CommonModule, CurrencyPipe, DatePipe, RouterLink],
  host: {
    class: 'flex-1 flex flex-col min-h-0 w-full bg-slate-50 dark:bg-slate-950 p-4 sm:p-8 overflow-y-auto',
  },
  template: `
    <div class="mx-auto space-y-6 w-full max-w-5xl">
      <!-- Loading State -->
      <div *ngIf="isLoading()" class="flex justify-center items-center py-20">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
      </div>

      <!-- Detail View -->
      <ng-container *ngIf="invoice() as inv">
        <!-- Header Actions -->
        <div class="flex items-center gap-4 mb-2">
          <a routerLink="../" class="inline-flex items-center text-sm font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300">
            <span class="material-icons-outlined text-[18px] mr-1">arrow_back</span>
            Volver a facturas
          </a>
        </div>

        <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
          <div>
            <h2 class="text-2xl font-bold leading-7 text-slate-900 dark:text-white sm:truncate sm:text-3xl sm:tracking-tight">Factura {{ inv.invoice_number || 'N/A' }}</h2>
            <p class="mt-1 text-sm leading-6 text-slate-500 dark:text-slate-400">Emitida el {{ inv.issue_date | date: 'longDate' }}</p>
          </div>
          <div class="flex items-center gap-3">
            <button class="btn-secondary">
              <span class="material-icons-outlined text-[18px] mr-1.5">download</span>
              Descargar
            </button>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mt-6">
          <!-- Issuer Info -->
          <div class="card !p-6 flex flex-col justify-between">
            <div>
              <h3 class="text-sm font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-4">Emisor</h3>
              <p class="text-lg font-semibold text-slate-900 dark:text-white mb-1">{{ inv.issuer_name || 'Desconocido' }}</p>
              <p class="text-sm text-slate-600 dark:text-slate-300">
                NIT: <span class="font-medium">{{ inv.issuer_tax_id || 'N/A' }}</span>
              </p>
            </div>
            <div class="mt-4 pt-4 border-t border-slate-200 dark:border-slate-800">
              <p class="text-sm text-slate-500 dark:text-slate-400">CUFE</p>
              <p class="text-xs text-slate-900 dark:text-white break-all mt-1 font-mono bg-slate-100 dark:bg-slate-800 p-2 rounded">{{ inv.cufe || 'N/A' }}</p>
            </div>
          </div>

          <!-- Receiver & Summary Info -->
          <div class="card !p-6 flex flex-col justify-between">
            <div>
              <h3 class="text-sm font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-4">Receptor</h3>
              <p class="text-lg font-semibold text-slate-900 dark:text-white mb-1">{{ inv.receiver_name || 'Desconocido' }}</p>
              <p class="text-sm text-slate-600 dark:text-slate-300">
                NIT: <span class="font-medium">{{ inv.receiver_tax_id || 'N/A' }}</span>
              </p>
            </div>
            <div class="mt-4 pt-4 border-t border-slate-200 dark:border-slate-800 grid grid-cols-2 gap-4">
              <div>
                <p class="text-sm text-slate-500 dark:text-slate-400">Vencimiento</p>
                <p class="text-sm font-medium text-slate-900 dark:text-white mt-1">{{ inv.due_date ? (inv.due_date | date: 'mediumDate') : 'N/A' }}</p>
              </div>
              <div>
                <p class="text-sm text-slate-500 dark:text-slate-400">Método de pago</p>
                <p class="text-sm font-medium text-slate-900 dark:text-white mt-1">{{ inv.payment_code || 'N/A' }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Lines Table -->
        <div class="card !p-0 mt-6 overflow-hidden shadow-sm">
          <div class="px-6 py-4 border-b border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900">
            <h3 class="text-lg font-medium leading-6 text-slate-900 dark:text-white">Detalle de Productos o Servicios</h3>
          </div>
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
              <thead class="bg-slate-50 dark:bg-slate-800/50">
                <tr>
                  <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Código</th>
                  <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Descripción</th>
                  <th scope="col" class="px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Cantidad</th>
                  <th scope="col" class="px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Precio Unitario</th>
                  <th scope="col" class="px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Impuesto</th>
                  <th scope="col" class="px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Total</th>
                </tr>
              </thead>
              <tbody class="bg-white dark:bg-slate-900 divide-y divide-slate-200 dark:divide-slate-800">
                <tr *ngFor="let line of inv.lines" class="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors">
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                    {{ line.item_code || 'N/A' }}
                  </td>
                  <td class="px-6 py-4 text-sm text-slate-900 dark:text-white">
                    <div class="font-medium max-w-[300px] break-words">{{ line.description }}</div>
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-right text-slate-500 dark:text-slate-400">
                    {{ line.quantity }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-right text-slate-500 dark:text-slate-400">
                    {{ line.unit_price | currency: inv.currency_code : 'symbol' : '1.2-2' }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-right text-slate-500 dark:text-slate-400">
                    {{ line.line_tax_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-right font-medium text-slate-900 dark:text-white">
                    {{ line.line_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}
                  </td>
                </tr>
                <tr *ngIf="!inv.lines || inv.lines.length === 0">
                  <td colspan="6" class="px-6 py-8 text-center text-sm text-slate-500 dark:text-slate-400">No se encontraron líneas de detalle para esta factura.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Totals Summary -->
        <div class="flex justify-end mt-6">
          <div class="card !p-0 w-full sm:w-80 overflow-hidden">
            <div class="px-6 py-4 border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
              <h3 class="text-sm font-medium text-slate-900 dark:text-white">Resumen de Totales</h3>
            </div>
            <div class="px-6 py-4 bg-white dark:bg-slate-900 space-y-3">
              <div class="flex justify-between items-center text-sm">
                <span class="text-slate-500 dark:text-slate-400">Subtotal</span>
                <span class="font-medium text-slate-900 dark:text-white">{{ inv.subtotal | currency: inv.currency_code : 'symbol' : '1.2-2' }}</span>
              </div>
              <div class="flex justify-between items-center text-sm">
                <span class="text-slate-500 dark:text-slate-400">Impuestos</span>
                <span class="font-medium text-slate-900 dark:text-white">{{ inv.tax_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}</span>
              </div>
              <div class="pt-3 border-t border-slate-200 dark:border-slate-800 flex justify-between items-center">
                <span class="text-base font-semibold text-slate-900 dark:text-white">Total a Pagar</span>
                <span class="text-lg font-bold text-indigo-600 dark:text-indigo-400">{{ inv.grand_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}</span>
              </div>
            </div>
          </div>
        </div>
      </ng-container>
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
