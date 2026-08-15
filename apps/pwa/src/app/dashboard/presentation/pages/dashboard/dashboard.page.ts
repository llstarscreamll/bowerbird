import { Component } from '@angular/core';
import { NgIcon } from '@ng-icons/core';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmEmptyImports } from '@spartan-ng/helm/empty';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [NgIcon, HlmCardImports, HlmEmptyImports],
  host: {
    class: 'flex-1 flex flex-col min-h-0 w-full',
  },
  template: `
    <div class="h-full w-full flex-1 overflow-y-auto p-8">
      <div class="mx-auto w-full space-y-6">
        <div>
          <h2 class="text-2xl font-bold tracking-tight sm:text-3xl">Dashboard</h2>
          <p class="mt-1 text-sm text-muted-foreground">Bienvenido a tu espacio de trabajo.</p>
        </div>

        <hlm-empty class="py-16">
          <ng-icon hlm name="lucideLayoutDashboard" class="text-6xl text-muted-foreground/40" />
          <h3 hlmEmptyTitle>Aún no hay datos para mostrar</h3>
          <p hlmEmptyDescription>Este es tu dashboard de inicio. Próximamente encontrarás aquí un resumen de tu actividad y métricas clave.</p>
        </hlm-empty>
      </div>
    </div>
  `,
})
export class DashboardPage {}
