import { Component } from '@angular/core';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [],
  host: {
    class: 'flex-1 flex flex-col min-h-0 w-full',
  },
  template: `
    <div class="h-full w-full bg-slate-50 dark:bg-slate-950 p-8 overflow-y-auto transition-colors duration-200 flex-1 flex flex-col">
      <div class="mx-auto space-y-6 w-full">
        <div>
          <h2 class="text-2xl font-bold leading-7 text-slate-900 dark:text-white sm:truncate sm:text-3xl sm:tracking-tight">Dashboard</h2>
          <p class="mt-1 text-sm leading-6 text-slate-500 dark:text-slate-400">Bienvenido a tu espacio de trabajo.</p>
        </div>

        <div class="card flex flex-col items-center justify-center py-16 text-center shadow-sm">
          <span class="material-icons-outlined text-slate-300 dark:text-slate-600 text-6xl mb-4 transition-colors">space_dashboard</span>
          <h3 class="text-lg font-medium text-slate-900 dark:text-white">Aún no hay datos para mostrar</h3>
          <p class="mt-2 text-sm text-slate-500 dark:text-slate-400 max-w-sm">Este es tu dashboard de inicio. Próximamente encontrarás aquí un resumen de tu actividad y métricas clave.</p>
        </div>
      </div>
    </div>
  `,
})
export class DashboardComponent {}
