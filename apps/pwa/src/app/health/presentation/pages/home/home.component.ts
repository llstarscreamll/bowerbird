import { ChangeDetectionStrategy, Component, OnInit, inject } from '@angular/core';
import { DatePipe } from '@angular/common';
import { HealthStore } from '../../../application/health.store';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [DatePipe],
  template: `
    <main class="mx-auto flex min-h-screen max-w-4xl flex-col items-center justify-center px-6">
      <section class="w-full rounded-2xl bg-white/80 p-8 shadow-xl backdrop-blur-sm">
        <p class="mb-2 text-sm uppercase tracking-[0.2em] text-cyan-600">Bowerbird Platform</p>
        <h1 class="mb-4 text-4xl font-bold">Angular SPA + Go API</h1>

        <div class="mb-6 flex items-center justify-between">
          <p class="text-slate-700">Estado del backend:</p>
          <button
            (click)="healthStore.checkHealth()"
            [disabled]="healthStore.isLoading()"
            class="text-xs font-semibold text-cyan-700 hover:text-cyan-900 disabled:opacity-50"
          >
            Refrescar
          </button>
        </div>

        <div class="inline-flex items-center gap-2 rounded-full bg-slate-100 px-4 py-2 text-sm">
          <span
            class="h-2.5 w-2.5 rounded-full"
            [class.bg-emerald-500]="healthStore.isHealthy()"
            [class.bg-orange-500]="!healthStore.isHealthy() && healthStore.status() !== 'checking...'"
            [class.bg-slate-400]="healthStore.status() === 'checking...'"
            [class.animate-pulse]="healthStore.isLoading()"
          ></span>
          <span>{{ healthStore.status() }}</span>
        </div>

        @if (healthStore.lastChecked()) {
          <p class="mt-4 text-xs text-slate-400">
            Última comprobación: {{ healthStore.lastChecked() | date: 'mediumTime' }}
          </p>
        }
      </section>
    </main>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HomeComponent implements OnInit {
  readonly healthStore = inject(HealthStore);

  ngOnInit(): void {
    this.healthStore.checkHealth();
  }
}
