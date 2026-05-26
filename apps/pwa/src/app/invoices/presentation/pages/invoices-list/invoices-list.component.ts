import { Component } from '@angular/core';

@Component({
  selector: 'app-invoices-list',
  standalone: true,
  imports: [],
  host: {
    class: 'flex-1 flex flex-col min-h-0 w-full',
  },
  template: `
    <div class="h-full w-full bg-slate-50 dark:bg-slate-950 p-8 overflow-y-auto transition-colors duration-200 flex-1 flex flex-col">
      <div class="mx-auto space-y-6 w-full">
        <!-- Header -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h2 class="text-2xl font-bold leading-7 text-slate-900 dark:text-white sm:truncate sm:text-3xl sm:tracking-tight">Facturas</h2>
            <p class="mt-1 text-sm leading-6 text-slate-500 dark:text-slate-400">Gestiona, filtra y revisa todas tus facturas electrónicas.</p>
          </div>
          <div class="flex items-center gap-3">
            <button class="btn-secondary">
              <span class="material-icons-outlined text-[18px] mr-1.5">filter_list</span>
              Filtrar
            </button>
            <button class="btn-primary">
              <span class="material-icons-outlined text-[18px] mr-1.5">add</span>
              Nueva Factura
            </button>
          </div>
        </div>

        <!-- Empty State Master -->
        <div class="card flex flex-col items-center justify-center py-20 text-center shadow-sm">
          <div class="w-20 h-20 bg-slate-100 dark:bg-slate-800 rounded-full flex items-center justify-center mb-6 shadow-[inset_0_2px_4px_rgba(0,0,0,0.02)] transition-colors">
            <span class="material-icons-outlined text-slate-300 dark:text-slate-600 text-4xl">receipt_long</span>
          </div>
          <h3 class="text-lg font-medium text-slate-900 dark:text-white">Aún no hay facturas</h3>
          <p class="mt-2 text-sm text-slate-500 dark:text-slate-400 max-w-sm mb-6">
            No se han encontrado facturas en este entorno. Pronto podrás sincronizarlas desde tu bandeja o crearlas manualmente.
          </p>
          <button class="btn-secondary">
            <span class="material-icons-outlined text-[18px] mr-1.5">cloud_download</span>
            Importar histórico
          </button>
        </div>
      </div>
    </div>
  `,
})
export class InvoicesListComponent {}
