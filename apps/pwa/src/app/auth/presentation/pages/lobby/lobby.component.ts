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
    <div class="min-h-screen bg-gray-100 py-12 px-4 sm:px-6 lg:px-8">
      <div class="max-w-3xl mx-auto">
        <div class="flex justify-between items-center mb-8">
          <h1 class="text-3xl font-bold text-gray-900">Welcome</h1>
          <button (click)="logout()" class="text-sm text-gray-600 hover:text-gray-900">Logout</button>
        </div>

        <div class="bg-white shadow overflow-hidden sm:rounded-lg">
          <div class="px-4 py-5 sm:px-6 flex justify-between items-center border-b border-gray-200">
            <h3 class="text-lg leading-6 font-medium text-gray-900">Your Organizations</h3>
            <button
              (click)="createNewTenant()"
              class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700"
            >
              Create New
            </button>
          </div>

          <div *ngIf="store.isLoading()" class="p-4 text-center text-gray-500">Loading...</div>

          <ul *ngIf="!store.isLoading() && store.tenants().length > 0" class="divide-y divide-gray-200">
            <li
              *ngFor="let tenant of store.tenants()"
              class="hover:bg-gray-50 cursor-pointer"
              (click)="selectTenant(tenant)"
            >
              <div class="px-4 py-4 sm:px-6 flex items-center justify-between">
                <div class="text-sm font-medium text-indigo-600 truncate">{{ tenant.tenant_id }}</div>
                <div class="ml-2 flex-shrink-0 flex">
                  <span
                    class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800"
                  >
                    {{ tenant.role }}
                  </span>
                </div>
              </div>
            </li>
          </ul>

          <div *ngIf="!store.isLoading() && store.tenants().length === 0" class="p-8 text-center text-gray-500">
            You don't belong to any organizations yet. Create one to get started!
          </div>
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
    // Navigate to tenant dashboard using path-based routing
    this.router.navigate(['/', tenant.tenant_id, 'dashboard']);
  }

  createNewTenant() {
    // In a real app, this would open a modal or go to a create page
    alert('Create tenant feature to be implemented in UI');
  }

  logout() {
    this.store.logout();
    setTimeout(() => {
      this.router.navigate(['/login']);
    }, 500);
  }
}
