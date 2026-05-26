import { ChangeDetectionStrategy, Component, OnInit, inject } from '@angular/core';
import { DatePipe, NgClass } from '@angular/common';
import { Router } from '@angular/router';
import { HealthStore } from '../../../application/health.store';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [DatePipe, NgClass],
  template: `
    <div class="min-h-screen flex flex-col">
      <!-- Top Navigation -->
      <nav class="bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800/80">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div class="flex justify-between h-16">
            <div class="flex items-center gap-3">
              <div class="h-8 w-8 bg-indigo-600 rounded-lg flex items-center justify-center shadow-sm">
                <span class="material-icons-outlined text-white text-lg">flight_takeoff</span>
              </div>
              <span class="font-semibold text-lg tracking-tight text-slate-900 dark:text-white">Bowerbird</span>
            </div>

            <div class="flex items-center gap-4">
              <button class="p-2 text-slate-400 hover:text-slate-500 dark:hover:text-slate-300 transition-colors">
                <span class="material-icons-outlined">notifications</span>
              </button>

              <div class="h-8 w-8 rounded-full bg-slate-200 dark:bg-slate-800 flex items-center justify-center border border-slate-300 dark:border-slate-700 cursor-pointer overflow-hidden">
                <span class="material-icons-outlined text-slate-500 text-sm">person</span>
              </div>
            </div>
          </div>
        </div>
      </nav>

      <!-- Main Content -->
      <main class="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <!-- Page Header -->
        <div class="md:flex md:items-center md:justify-between mb-8">
          <div class="min-w-0 flex-1">
            <h2 class="text-2xl font-bold leading-7 text-slate-900 dark:text-white sm:truncate sm:text-3xl sm:tracking-tight">Dashboard Overview</h2>
          </div>
          <div class="mt-4 flex md:ml-4 md:mt-0 gap-3">
            <button (click)="goLobby()" class="btn-secondary gap-2">
              <span class="material-icons-outlined text-sm">domain</span>
              Switch Organization
            </button>
            <button (click)="healthStore.checkHealth()" [disabled]="healthStore.isLoading()" class="btn-primary gap-2">
              <span class="material-icons-outlined text-sm" [class.animate-spin]="healthStore.isLoading()">refresh</span>
              Refresh Status
            </button>
          </div>
        </div>

        <!-- Metrics Grid -->
        <div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          <!-- System Health Card -->
          <div class="card flex flex-col justify-between">
            <div>
              <div class="flex items-center justify-between">
                <p class="text-sm font-medium text-slate-500 dark:text-slate-400 truncate">API Health Status</p>
                <div class="p-2 bg-indigo-50 dark:bg-indigo-500/10 rounded-lg">
                  <span class="material-icons-outlined text-indigo-600 dark:text-indigo-400 text-sm">dns</span>
                </div>
              </div>

              <div class="mt-4 flex items-center gap-3">
                <span class="relative flex h-4 w-4">
                  @if (healthStore.isHealthy() && !healthStore.isLoading()) {
                    <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                  }
                  <span
                    class="relative inline-flex rounded-full h-4 w-4"
                    [ngClass]="{
                      'bg-emerald-500': healthStore.isHealthy() && !healthStore.isLoading(),
                      'bg-amber-500': !healthStore.isHealthy() && healthStore.status() !== 'checking...' && !healthStore.isLoading(),
                      'bg-slate-400': healthStore.status() === 'checking...' || healthStore.isLoading(),
                    }"
                  ></span>
                </span>

                <p class="text-2xl font-semibold tracking-tight text-slate-900 dark:text-white">
                  {{ healthStore.isLoading() ? 'Checking...' : healthStore.isHealthy() ? 'Operational' : 'Degraded' }}
                </p>
              </div>
            </div>

            <div class="mt-6 text-sm">
              <p class="text-slate-500 dark:text-slate-400 flex items-center gap-1">
                <span class="material-icons-outlined text-xs">schedule</span>
                Last checked:
                <span class="font-medium text-slate-700 dark:text-slate-300">
                  {{ healthStore.lastChecked() ? (healthStore.lastChecked() | date: 'shortTime') : 'Never' }}
                </span>
              </p>
            </div>
          </div>

          <!-- Placeholder Metric 1 -->
          <div class="card flex flex-col justify-between">
            <div>
              <div class="flex items-center justify-between">
                <p class="text-sm font-medium text-slate-500 dark:text-slate-400 truncate">Active Users</p>
                <div class="p-2 bg-emerald-50 dark:bg-emerald-500/10 rounded-lg">
                  <span class="material-icons-outlined text-emerald-600 dark:text-emerald-400 text-sm">people</span>
                </div>
              </div>
              <div class="mt-4 flex items-baseline gap-2">
                <p class="text-3xl font-semibold tracking-tight text-slate-900 dark:text-white">1,204</p>
                <p class="text-sm font-medium text-emerald-600 dark:text-emerald-400 flex items-center">
                  <span class="material-icons-outlined text-xs">arrow_upward</span>
                  12%
                </p>
              </div>
            </div>
            <div class="mt-6 text-sm text-slate-500 dark:text-slate-400">Compared to last week</div>
          </div>

          <!-- Placeholder Metric 2 -->
          <div class="card flex flex-col justify-between">
            <div>
              <div class="flex items-center justify-between">
                <p class="text-sm font-medium text-slate-500 dark:text-slate-400 truncate">Compute Usage</p>
                <div class="p-2 bg-amber-50 dark:bg-amber-500/10 rounded-lg">
                  <span class="material-icons-outlined text-amber-600 dark:text-amber-400 text-sm">memory</span>
                </div>
              </div>
              <div class="mt-4 flex items-baseline gap-2">
                <p class="text-3xl font-semibold tracking-tight text-slate-900 dark:text-white">42%</p>
                <p class="text-sm font-medium text-slate-500 dark:text-slate-500 flex items-center">Stable</p>
              </div>
            </div>

            <div class="mt-6 w-full bg-slate-100 dark:bg-slate-800 rounded-full h-1.5 mb-1 overflow-hidden">
              <div class="bg-amber-500 h-1.5 rounded-full" style="width: 42%"></div>
            </div>
          </div>
        </div>
      </main>
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HomeComponent implements OnInit {
  readonly healthStore = inject(HealthStore);
  private router = inject(Router);

  ngOnInit(): void {
    this.healthStore.checkHealth();
  }

  goLobby() {
    this.router.navigate(['/lobby']);
  }
}
