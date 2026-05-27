import { CommonModule } from '@angular/common';
import { Component, Input, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { ConnectionsStore } from '../../../application/connections.store';
import { TenantContextStore } from '../../../../core/store/tenant-context.store';
import { Connection, ConnectionProvider, ConnectionStatus } from '../../../domain/connections.model';
import { ModalComponent } from '../../../../core/presentation/components/modal/modal.component';
import { IconGoogleComponent } from '../../../../core/presentation/components/icons/icon-google.component';
import { IconMicrosoftComponent } from '../../../../core/presentation/components/icons/icon-microsoft.component';

@Component({
  selector: 'app-account-status-chip',
  standalone: true,
  imports: [CommonModule],
  template: `
    <span
      class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium"
      [ngClass]="{
        'bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-600/20': status === 'active',
        'bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-600/20': status === 'requires_reconnect',
        'bg-slate-50 text-slate-700 ring-1 ring-inset ring-slate-600/20': status === 'paused',
        'bg-rose-50 text-rose-700 ring-1 ring-inset ring-rose-600/20': status === 'error',
      }"
    >
      {{ label }}
    </span>
  `,
})
export class AccountStatusChipComponent {
  @Input() status: ConnectionStatus = 'active';

  get label(): string {
    switch (this.status) {
      case 'active':
        return 'Activa';
      case 'requires_reconnect':
        return 'Requiere reconexión';
      case 'paused':
        return 'Pausada';
      case 'error':
        return 'Error';
      default:
        return this.status;
    }
  }
}

@Component({
  selector: 'app-connections-list',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule, AccountStatusChipComponent, ModalComponent, IconGoogleComponent, IconMicrosoftComponent],
  host: {
    class: 'flex-1 flex flex-col min-h-0 w-full',
  },
  template: `
    <div class="h-full bg-gradient-to-b from-slate-50 to-white dark:from-slate-950 dark:to-slate-900 px-4 py-8 sm:px-6 lg:px-8 overflow-y-auto transition-colors duration-200">
      <div class="mx-auto w-full max-w-5xl space-y-8">
        <!-- Encabezado de la página -->
        <header class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 class="text-2xl font-semibold tracking-tight text-slate-900 sm:text-3xl">Conexiones</h1>
            <p class="mt-1 max-w-2xl text-sm text-slate-500">
              Gestiona las cuentas de correo vinculadas a la organización <span class="font-medium text-slate-700">{{ tenantName()?.name || 'actual' }}</span
              >.
            </p>
          </div>
          <button class="btn-primary flex items-center gap-2" (click)="openConnectModal()">
            <span class="material-icons-outlined text-[18px]">add</span>
            <span>Añadir cuenta</span>
          </button>
        </header>

        <!-- Listado principal de conexiones -->
        <div class="card p-0 overflow-hidden">
          <div class="border-b border-slate-200 bg-slate-50/50 px-6 py-4 flex items-center justify-between">
            <h2 class="text-sm font-semibold uppercase tracking-wide text-slate-700">Cuentas vinculadas</h2>
            <div class="flex gap-2 text-xs">
              <span *ngIf="statusCount('active') > 0" class="rounded-full bg-emerald-100 px-2.5 py-0.5 font-medium text-emerald-800"> {{ statusCount('active') }} Activas </span>
              <span *ngIf="statusCount('requires_reconnect') > 0" class="rounded-full bg-amber-100 px-2.5 py-0.5 font-medium text-amber-800"> {{ statusCount('requires_reconnect') }} Problemas </span>
            </div>
          </div>

          <div *ngIf="loading()" class="px-6 py-12 text-center text-sm text-slate-500">
            <div class="inline-block h-6 w-6 animate-spin rounded-full border-2 border-slate-300 border-t-indigo-600 mb-2"></div>
            <p>Cargando conexiones...</p>
          </div>

          <div *ngIf="!loading() && connections().length === 0" class="px-6 py-16 text-center">
            <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 mb-4">
              <span class="material-icons-outlined text-slate-400">link_off</span>
            </div>
            <h3 class="text-sm font-semibold text-slate-900">No hay cuentas conectadas</h3>
            <p class="mt-1 text-sm text-slate-500">Añade tu primera cuenta de correo para empezar a sincronizar facturas y comprobantes.</p>
            <button class="mt-6 btn-secondary" (click)="openConnectModal()">Añadir primera cuenta</button>
          </div>

          <ul *ngIf="!loading() && connections().length > 0" class="divide-y divide-slate-100">
            <li *ngFor="let conn of connections()" class="group relative px-6 py-5 hover:bg-slate-50 transition-colors">
              <div class="flex items-start justify-between gap-4">
                <div class="flex items-center gap-4">
                  <!-- Logo del proveedor -->
                  <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white shadow-sm p-2.5">
                    <app-icon-google *ngIf="conn.provider === 'gmail'" class="h-full w-full object-contain"></app-icon-google>
                    <app-icon-microsoft *ngIf="conn.provider === 'microsoft'" class="h-full w-full object-contain"></app-icon-microsoft>
                  </div>

                  <!-- Info -->
                  <div class="min-w-0">
                    <div class="flex items-center gap-2">
                      <p class="text-sm font-semibold text-slate-900 truncate">{{ conn.provider_account_email }}</p>
                      <app-account-status-chip [status]="conn.status" />
                    </div>
                    <div class="mt-1 flex items-center gap-3 text-xs text-slate-500">
                      <span>{{ providerLabel(conn.provider) }}</span>
                      <span class="h-1 w-1 rounded-full bg-slate-300"></span>
                      <span class="flex items-center gap-1">
                        <span class="material-icons-outlined text-[14px]">visibility</span>
                        {{ conn.sharing_policy === 'private' ? 'Privado (Solo yo)' : 'Compartido (Equipo)' }}
                      </span>
                    </div>
                  </div>
                </div>

                <!-- Acciones -->
                <div class="flex items-center gap-2 opacity-0 transition-opacity group-hover:opacity-100">
                  <button
                    class="rounded p-1.5 text-slate-400 hover:bg-slate-100 hover:text-slate-700 transition-colors"
                    title="Detalles y configuración"
                    [routerLink]="['/', tenantId(), 'connections', conn.id]"
                  >
                    <span class="material-icons-outlined text-[18px]">settings</span>
                  </button>
                </div>
              </div>
            </li>
          </ul>
        </div>
      </div>
    </div>

    <!-- Modal para añadir nueva cuenta -->
    <app-modal [isOpen]="isConnectModalOpen" title="Añadir nueva cuenta" (close)="closeConnectModal()">
      <div class="space-y-6">
        <p class="text-slate-600 leading-relaxed">Selecciona tu proveedor de correo para vincular tu cuenta. Serás redirigido a una página segura para autorizar el acceso.</p>

        <div class="rounded-lg bg-indigo-50 p-4">
          <h4 class="flex items-center gap-2 text-sm font-semibold text-indigo-900">
            <span class="material-icons-outlined text-[18px]">security</span>
            Permisos que solicitaremos
          </h4>
          <ul class="mt-2 space-y-1 text-sm text-indigo-700 list-disc list-inside">
            <li>Leer correos electrónicos (para encontrar facturas)</li>
            <li>Crear y asignar etiquetas (para organizar tu bandeja)</li>
          </ul>
          <p class="mt-3 text-xs text-indigo-600/80">Nunca borraremos tus correos ni enviaremos mensajes en tu nombre.</p>
        </div>

        <div class="grid gap-3 pt-2">
          <button
            class="flex items-center justify-center gap-3 rounded-lg border border-slate-300 bg-white px-4 py-3 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 transition-all disabled:opacity-50"
            (click)="connect('gmail')"
            [disabled]="submitting()"
          >
            <div class="h-5 w-5">
              <app-icon-google></app-icon-google>
            </div>
            Continuar con Google
          </button>

          <button
            class="flex items-center justify-center gap-3 rounded-lg border border-slate-300 bg-white px-4 py-3 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 transition-all disabled:opacity-50 opacity-50 cursor-not-allowed"
            disabled
            title="Próximamente"
          >
            <div class="h-5 w-5">
              <app-icon-microsoft></app-icon-microsoft>
            </div>
            Continuar con Microsoft (Pronto)
          </button>
        </div>
      </div>
    </app-modal>
  `,
})
export class ConnectionsListComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  readonly store = inject(ConnectionsStore);
  private readonly tenantContext = inject(TenantContextStore);

  readonly providers = this.store.providers;
  readonly connections = this.store.connections;
  readonly loading = this.store.loading;
  readonly submitting = this.store.submitting;
  readonly disconnectingId = this.store.disconnectingId;
  readonly errorMessage = this.store.errorMessage;

  readonly tenantId = this.tenantContext.tenantId;
  readonly tenantName = this.tenantContext.tenantDetails;

  isConnectModalOpen = false;

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('tenantId');
    if (id) {
      this.tenantContext.setTenantId(id);
    }
    this.store.loadConnections();
  }

  openConnectModal(): void {
    this.isConnectModalOpen = true;
  }

  closeConnectModal(): void {
    this.isConnectModalOpen = false;
  }

  connect(provider: ConnectionProvider): void {
    this.store.connectProvider(provider, (authURL) => window.location.assign(authURL));
  }

  disconnect(conn: Connection): void {
    const confirmed = window.confirm(`Se desvinculará la cuenta ${conn.provider_account_email}. ¿Deseas continuar?`);
    if (!confirmed) {
      return;
    }
    this.store.disconnectConnection(conn.id);
  }

  providerLabel(provider: ConnectionProvider): string {
    return this.store.providerLabel(provider);
  }

  statusCount(status: ConnectionStatus): number {
    return this.connections().filter((conn) => conn.status === status).length;
  }
}
