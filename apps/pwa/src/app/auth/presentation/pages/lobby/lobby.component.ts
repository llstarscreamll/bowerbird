import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AuthStore } from '../../../application/auth.store';
import { TenantMembership } from '../../../domain/auth.model';

@Component({
  selector: 'app-lobby',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="min-h-screen py-12 px-4 sm:px-6 lg:px-8">
      <div class="max-w-3xl mx-auto space-y-8">
        <!-- Header -->
        <header class="flex justify-between items-end">
          <div>
            <h1 class="text-3xl font-semibold tracking-tight text-slate-900 dark:text-white">Welcome</h1>
            <p class="mt-2 text-sm text-slate-600 dark:text-slate-400">Select an organization to continue</p>
          </div>
          <button (click)="logout()" class="btn-secondary gap-2 text-slate-600 dark:text-slate-300">
            <span class="material-icons-outlined text-sm">logout</span>
            Logout
          </button>
        </header>

        <!-- Organizations Card -->
        <div class="card p-0 sm:p-0 overflow-hidden">
          <!-- Card Header -->
          <div
            class="px-6 py-5 border-b border-slate-200 dark:border-slate-800/80 flex justify-between items-center bg-slate-50/50 dark:bg-slate-900/50"
          >
            <h3 class="text-base font-semibold leading-6 text-slate-900 dark:text-white">Your Organizations</h3>
            <button (click)="createNewTenant()" class="btn-primary py-2 px-3 text-xs gap-1">
              <span class="material-icons-outlined text-sm">add</span>
              Create New
            </button>
          </div>

          <!-- Loading State -->
          <div *ngIf="store.isLoading()" class="p-12 flex flex-col items-center justify-center space-y-4">
            <svg class="animate-spin h-8 w-8 text-indigo-600 dark:text-indigo-500" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            <span class="text-sm text-slate-500 dark:text-slate-400">Loading organizations...</span>
          </div>

          <!-- Empty State -->
          <div *ngIf="!store.isLoading() && store.tenants().length === 0" class="p-16 text-center">
            <div
              class="mx-auto h-12 w-12 bg-slate-100 dark:bg-slate-800 rounded-full flex items-center justify-center mb-4"
            >
              <span class="material-icons-outlined text-slate-400 dark:text-slate-500">domain_disabled</span>
            </div>
            <h3 class="text-sm font-medium text-slate-900 dark:text-white">No organizations found</h3>
            <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">
              You don't belong to any organizations yet. Create one to get started!
            </p>
          </div>

          <!-- Tenant List -->
          <ul
            *ngIf="!store.isLoading() && store.tenants().length > 0"
            class="divide-y divide-slate-200 dark:divide-slate-800/80"
          >
            <li
              *ngFor="let tenant of store.tenants()"
              class="group relative hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors duration-200"
              (click)="selectTenant(tenant)"
            >
              <div class="px-6 py-5 flex items-center justify-between">
                <div class="flex items-center gap-4">
                  <!-- Avatar placeholder -->
                  <div
                    class="h-10 w-10 flex-shrink-0 bg-indigo-100 dark:bg-indigo-900/50 text-indigo-600 dark:text-indigo-400 rounded-lg flex items-center justify-center font-semibold text-sm border border-indigo-200 dark:border-indigo-800"
                  >
                    {{ tenant.tenant_id.substring(0, 2).toUpperCase() }}
                  </div>
                  <div>
                    <div
                      class="text-sm font-semibold text-slate-900 dark:text-white group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors"
                    >
                      {{ tenant.tenant_id }}
                    </div>
                    <div class="text-xs text-slate-500 dark:text-slate-400 mt-0.5">Workspace</div>
                  </div>
                </div>

                <div class="flex items-center gap-4">
                  <span
                    class="px-2.5 py-1 inline-flex text-xs font-medium rounded-md border"
                    [ngClass]="{
                      'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20':
                        tenant.role === 'OWNER',
                      'bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-500/10 dark:text-blue-400 dark:border-blue-500/20':
                        tenant.role === 'ADMIN',
                      'bg-slate-50 text-slate-700 border-slate-200 dark:bg-slate-500/10 dark:text-slate-400 dark:border-slate-500/20':
                        tenant.role !== 'OWNER' && tenant.role !== 'ADMIN',
                    }"
                  >
                    {{ tenant.role }}
                  </span>
                  <span class="material-icons-outlined text-slate-400 group-hover:text-indigo-500 transition-colors">
                    chevron_right
                  </span>
                </div>
              </div>
            </li>
          </ul>
        </div>
      </div>
    </div>
  `,
})
export class LobbyComponent implements OnInit {
  readonly store = inject(AuthStore);
  private router = inject(Router);

  ngOnInit() {
    if (!this.store.isAuthenticated()) {
      this.router.navigate(['/login']);
      return;
    }
    this.store.loadTenants();
  }

  selectTenant(tenant: TenantMembership) {
    this.router.navigate(['/', tenant.tenant_id, 'dashboard']);
  }

  createNewTenant() {
    alert('Create tenant feature to be implemented in UI');
  }

  logout() {
    this.store.logout();
    setTimeout(() => {
      this.router.navigate(['/login']);
    }, 500);
  }
}
