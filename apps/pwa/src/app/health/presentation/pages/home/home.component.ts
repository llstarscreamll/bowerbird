import { ChangeDetectionStrategy, Component, OnInit, inject } from '@angular/core';
import { DatePipe, NgClass } from '@angular/common';
import { Router } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HealthStore } from '../../../application/health.store';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [DatePipe, NgClass, NgIcon, HlmCardImports, HlmButtonImports, HlmSpinnerImports],
  template: `
    <div class="flex min-h-screen flex-col">
      <nav class="border-b border-border bg-card">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="flex h-16 justify-between">
            <div class="flex items-center gap-3">
              <div class="flex size-8 items-center justify-center rounded-lg bg-primary shadow-sm">
                <ng-icon name="lucidePlaneTakeoff" class="text-primary-foreground" />
              </div>
              <span class="text-lg font-semibold tracking-tight">Bowerbird</span>
            </div>
            <div class="flex items-center gap-4">
              <button hlmBtn variant="ghost" size="icon">
                <ng-icon name="lucideBell" />
              </button>
              <div class="flex size-8 items-center justify-center overflow-hidden rounded-full border bg-muted">
                <ng-icon name="lucideUser" class="text-muted-foreground" />
              </div>
            </div>
          </div>
        </div>
      </nav>

      <main class="mx-auto w-full max-w-7xl flex-1 px-4 py-8 sm:px-6 lg:px-8">
        <div class="mb-8 md:flex md:items-center md:justify-between">
          <h2 class="text-2xl font-bold tracking-tight sm:text-3xl">Dashboard Overview</h2>
          <div class="mt-4 flex gap-3 md:ml-4 md:mt-0">
            <button hlmBtn variant="outline" (click)="goLobby()">
              <ng-icon name="lucideBuilding2" />
              Switch Organization
            </button>
            <button hlmBtn (click)="healthStore.checkHealth()" [disabled]="healthStore.isLoading()">
              <ng-icon name="lucideRefreshCw" [class.animate-spin]="healthStore.isLoading()" />
              Refresh Status
            </button>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          <hlm-card class="flex flex-col justify-between p-6">
            <div>
              <div class="flex items-center justify-between">
                <p class="truncate text-sm font-medium text-muted-foreground">API Health Status</p>
                <div class="rounded-lg bg-primary/10 p-2">
                  <ng-icon name="lucideServer" class="text-primary" />
                </div>
              </div>
              <div class="mt-4 flex items-center gap-3">
                <span class="relative flex size-4">
                  @if (healthStore.isHealthy() && !healthStore.isLoading()) {
                    <span class="absolute inline-flex size-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
                  }
                  <span
                    class="relative inline-flex size-4 rounded-full"
                    [ngClass]="{
                      'bg-emerald-500': healthStore.isHealthy() && !healthStore.isLoading(),
                      'bg-amber-500': !healthStore.isHealthy() && healthStore.status() !== 'checking...' && !healthStore.isLoading(),
                      'bg-muted-foreground': healthStore.status() === 'checking...' || healthStore.isLoading(),
                    }"
                  ></span>
                </span>
                <p class="text-2xl font-semibold tracking-tight">
                  {{ healthStore.isLoading() ? 'Checking...' : healthStore.isHealthy() ? 'Operational' : 'Degraded' }}
                </p>
              </div>
            </div>
            <p class="mt-6 flex items-center gap-1 text-sm text-muted-foreground">
              <ng-icon name="lucideClock" class="text-xs" />
              Last checked:
              <span class="font-medium text-foreground">{{ healthStore.lastChecked() ? (healthStore.lastChecked() | date: 'shortTime') : 'Never' }}</span>
            </p>
          </hlm-card>

          <hlm-card class="flex flex-col justify-between p-6">
            <div>
              <div class="flex items-center justify-between">
                <p class="truncate text-sm font-medium text-muted-foreground">Active Users</p>
                <div class="rounded-lg bg-emerald-500/10 p-2">
                  <ng-icon name="lucideUsers" class="text-emerald-600 dark:text-emerald-400" />
                </div>
              </div>
              <div class="mt-4 flex items-baseline gap-2">
                <p class="text-3xl font-semibold tracking-tight">1,204</p>
                <p class="flex items-center text-sm font-medium text-emerald-600 dark:text-emerald-400">
                  <ng-icon name="lucideArrowUp" class="text-xs" />
                  12%
                </p>
              </div>
            </div>
            <div class="mt-6 text-sm text-muted-foreground">Compared to last week</div>
          </hlm-card>

          <hlm-card class="flex flex-col justify-between p-6">
            <div>
              <div class="flex items-center justify-between">
                <p class="truncate text-sm font-medium text-muted-foreground">Compute Usage</p>
                <div class="rounded-lg bg-amber-500/10 p-2">
                  <ng-icon name="lucideCpu" class="text-amber-600 dark:text-amber-400" />
                </div>
              </div>
              <div class="mt-4 flex items-baseline gap-2">
                <p class="text-3xl font-semibold tracking-tight">42%</p>
                <p class="text-sm font-medium text-muted-foreground">Stable</p>
              </div>
            </div>
            <div class="mt-6 mb-1 h-1.5 w-full overflow-hidden rounded-full bg-muted">
              <div class="h-1.5 rounded-full bg-amber-500" style="width: 42%"></div>
            </div>
          </hlm-card>
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
