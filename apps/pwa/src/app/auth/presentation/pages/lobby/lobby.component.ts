import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AuthStore } from '../../../application/auth.store';
import { LobbyStore } from '../../../application/lobby.store';
import { TenantMembership } from '../../../domain/auth.model';
import { AlertComponent } from '../../../../core/presentation/components/alert/alert.component';

@Component({
  selector: 'app-lobby',
  standalone: true,
  imports: [CommonModule, FormsModule, AlertComponent],
  template: `
    <div class="min-h-screen py-10 px-4 sm:px-6 lg:px-8 bg-slate-50/50 dark:bg-slate-950/50">
      <div class="max-w-5xl mx-auto space-y-6">
        <!-- Header -->
        <header class="flex justify-between items-center pb-4 border-b border-slate-200 dark:border-slate-800/80">
          <div>
            <h1 class="text-2xl font-semibold tracking-tight text-slate-900 dark:text-white">Bienvenido</h1>
            <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">Selecciona una organización para continuar</p>
          </div>
          <button (click)="logout()" class="btn-secondary gap-2 text-slate-700 dark:text-slate-300 py-1.5 px-3">
            <span class="material-icons-outlined text-[18px]">logout</span>
            <span class="hidden sm:inline">Cerrar sesión</span>
          </button>
        </header>

        <!-- Organizations Card -->
        <div class="card p-0 sm:p-0 overflow-hidden shadow-sm">
          <!-- Card Header -->
          <div class="px-5 py-4 border-b border-slate-200 dark:border-slate-800/80 flex justify-between items-center bg-white dark:bg-slate-900">
            <h3 class="text-sm font-medium leading-6 text-slate-900 dark:text-white">Tus Organizaciones</h3>
            <button (click)="toggleCreateForm()" class="btn-primary py-1.5 px-3 gap-2">
              <span class="material-icons-outlined text-[18px]">{{ showCreateForm() ? 'close' : 'add' }}</span>
              <span>{{ showCreateForm() ? 'Cancelar' : 'Crear nueva' }}</span>
            </button>
          </div>

          <!-- Create Tenant Form -->
          <div *ngIf="showCreateForm()" class="p-5 bg-slate-50/50 dark:bg-slate-800/20 border-b border-slate-200 dark:border-slate-800/80">
            <form (ngSubmit)="onCreateTenant()" class="space-y-4">
              <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
                <div>
                  <label for="orgName" class="block text-sm font-medium text-slate-700 dark:text-slate-300"> Nombre de la Organización </label>
                  <div class="mt-1.5">
                    <input id="orgName" type="text" required [ngModel]="newOrgName()" (ngModelChange)="onOrgNameInput($event)" name="orgName" class="input-field py-2" placeholder="Acme Corp" />
                  </div>
                </div>
                <div>
                  <label for="orgSlug" class="block text-sm font-medium text-slate-700 dark:text-slate-300"> URL del espacio / Slug </label>
                  <div class="mt-1.5">
                    <input id="orgSlug" type="text" required [ngModel]="newOrgSlug()" (ngModelChange)="setNewOrgSlug($event)" name="orgSlug" class="input-field py-2" placeholder="acme" />
                  </div>
                </div>
              </div>

              <app-alert *ngIf="createError()" type="error">
                {{ createError() }}
              </app-alert>

              <div class="flex justify-end pt-2">
                <button type="submit" [disabled]="isCreating()" class="btn-primary py-2 px-4 shadow-sm text-sm">
                  <span *ngIf="!isCreating()">Crear Organización</span>
                  <span *ngIf="isCreating()" class="flex items-center gap-2">
                    <svg class="animate-spin h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    Creando...
                  </span>
                </button>
              </div>
            </form>
          </div>

          <!-- Loading State -->
          <div *ngIf="store.isLoading()" class="py-12 flex flex-col items-center justify-center space-y-3">
            <svg class="animate-spin h-6 w-6 text-indigo-600 dark:text-indigo-500" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <span class="text-sm text-slate-500 dark:text-slate-400">Cargando organizaciones...</span>
          </div>

          <!-- Empty State -->
          <div *ngIf="!store.isLoading() && store.tenants().length === 0" class="py-16 px-6 text-center">
            <div class="mx-auto h-10 w-10 bg-slate-100 dark:bg-slate-800 rounded-full flex items-center justify-center mb-4">
              <span class="material-icons-outlined text-slate-400 dark:text-slate-500">domain</span>
            </div>
            <h3 class="text-sm font-medium text-slate-900 dark:text-white">No se encontraron organizaciones</h3>
            <p class="mt-1 text-sm text-slate-500 dark:text-slate-400 max-w-sm mx-auto">Aún no perteneces a ninguna organización. Crea una nueva para comenzar.</p>
          </div>

          <!-- Tenant List -->
          <ul *ngIf="!store.isLoading() && store.tenants().length > 0" class="divide-y divide-slate-100 dark:divide-slate-800/60 bg-white dark:bg-slate-900">
            <li
              *ngFor="let tenant of store.tenants()"
              class="group relative hover:bg-slate-50/80 dark:hover:bg-slate-800/30 cursor-pointer transition-colors duration-200"
              (click)="selectTenant(tenant)"
            >
              <div class="px-5 py-4 flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <!-- Avatar placeholder -->
                  <div
                    class="h-9 w-9 flex-shrink-0 bg-indigo-50 dark:bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 rounded-lg flex items-center justify-center font-medium text-xs border border-indigo-100/50 dark:border-indigo-500/20 uppercase"
                  >
                    {{ (tenant.name || tenant.tenant_id).substring(0, 2) }}
                  </div>
                  <div>
                    <div class="text-sm font-medium text-slate-900 dark:text-slate-200 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
                      {{ tenant.name || tenant.tenant_id }}
                    </div>
                  </div>
                </div>

                <div class="flex items-center gap-4">
                  <button class="btn-secondary px-2.5 py-1.5 text-xs" (click)="goToConnections($event, tenant)" type="button">Correo</button>
                  <button class="btn-secondary px-2.5 py-1.5 text-xs" (click)="goToUnifiedInbox($event, tenant)" type="button">Bandeja</button>
                  <span
                    class="px-2 py-0.5 inline-flex text-[11px] font-medium rounded-md border"
                    [ngClass]="{
                      'bg-slate-50 text-slate-600 border-slate-200 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-700': tenant.role === 'OWNER',
                      'bg-slate-50 text-slate-500 border-slate-200 dark:bg-slate-800/50 dark:text-slate-400 dark:border-slate-700/50': tenant.role !== 'OWNER',
                    }"
                  >
                    {{ tenant.role | titlecase }}
                  </span>
                  <span class="material-icons-outlined text-slate-300 dark:text-slate-600 group-hover:text-indigo-500 dark:group-hover:text-indigo-400 transition-colors text-[20px]">
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
  private readonly lobbyStore = inject(LobbyStore);

  readonly showCreateForm = this.lobbyStore.showCreateForm;
  readonly newOrgName = this.lobbyStore.newOrgName;
  readonly newOrgSlug = this.lobbyStore.newOrgSlug;
  readonly isCreating = this.lobbyStore.isCreating;
  readonly createError = this.lobbyStore.createError;

  ngOnInit() {
    this.lobbyStore.init();
  }

  selectTenant(tenant: TenantMembership) {
    this.lobbyStore.selectTenant(tenant);
  }

  goToConnections(event: MouseEvent, tenant: TenantMembership) {
    event.stopPropagation();
    this.lobbyStore.goToConnections(tenant);
  }

  goToUnifiedInbox(event: MouseEvent, tenant: TenantMembership) {
    event.stopPropagation();
    this.lobbyStore.goToUnifiedInbox(tenant);
  }

  toggleCreateForm() {
    this.lobbyStore.toggleCreateForm();
  }

  onNameChange(name: string) {
    this.lobbyStore.onNameChange(name);
  }

  onOrgNameInput(name: string): void {
    this.setNewOrgName(name);
    this.onNameChange(name);
  }

  onCreateTenant() {
    this.lobbyStore.createTenant();
  }

  logout() {
    this.lobbyStore.logout();
  }

  setNewOrgName(name: string): void {
    this.lobbyStore.setNewOrgName(name);
  }

  setNewOrgSlug(slug: string): void {
    this.lobbyStore.setNewOrgSlug(slug);
  }
}
