import { CommonModule, Location } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { ConnectionsStore } from '../../../application/connections.store';
import { ConnectionProvider, ConnectionStatus } from '../../../domain/connections.model';
import { TenantContextStore } from '../../../../core/store/tenant-context.store';
import { ModalComponent } from '../../../../core/presentation/components/modal/modal.component';
import { IconGoogleComponent } from '../../../../core/presentation/components/icons/icon-google.component';
import { IconMicrosoftComponent } from '../../../../core/presentation/components/icons/icon-microsoft.component';
import { AccountStatusChipComponent } from '../connections-list/connections-list.component';

@Component({
  selector: 'app-connection-details',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule, ModalComponent, IconGoogleComponent, IconMicrosoftComponent, AccountStatusChipComponent],
  host: {
    class: 'flex-1 flex flex-col min-h-0 w-full',
  },
  template: `
    <div class="h-full bg-gradient-to-b from-slate-50 to-white dark:from-slate-950 dark:to-slate-900 px-4 py-8 sm:px-6 lg:px-8 overflow-y-auto transition-colors duration-200">
      <div class="mx-auto w-full max-w-3xl space-y-8">
        <!-- Header -->
        <header class="flex items-center gap-4">
          <button class="flex items-center justify-center rounded-full p-2 text-slate-500 hover:bg-slate-100 hover:text-slate-900 transition-colors" (click)="goBack()">
            <span class="material-icons-outlined">arrow_back</span>
          </button>
          <div>
            <h1 class="text-2xl font-semibold tracking-tight text-slate-900 sm:text-3xl">Detalles de la Conexión</h1>
            <p class="mt-1 text-sm text-slate-500">Configura las preferencias y visibilidad para esta cuenta.</p>
          </div>
        </header>

        <div *ngIf="loading() && !connection()" class="px-6 py-12 text-center text-sm text-slate-500 card">
          <div class="inline-block h-6 w-6 animate-spin rounded-full border-2 border-slate-300 border-t-indigo-600 mb-2"></div>
          <p>Cargando detalles...</p>
        </div>

        <div *ngIf="connection() as conn" class="space-y-6">
          <!-- Tarjeta de Información Principal -->
          <div class="card p-6">
            <div class="flex items-start justify-between gap-4">
              <div class="flex items-center gap-4">
                <div class="flex h-16 w-16 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white shadow-sm p-3">
                  <app-icon-google *ngIf="conn.provider === 'gmail'" class="h-full w-full object-contain"></app-icon-google>
                  <app-icon-microsoft *ngIf="conn.provider === 'microsoft'" class="h-full w-full object-contain"></app-icon-microsoft>
                </div>
                <div>
                  <h2 class="text-xl font-semibold text-slate-900">{{ conn.provider_account_email }}</h2>
                  <div class="mt-2 flex items-center gap-2">
                    <app-account-status-chip [status]="conn.status" />
                    <span class="text-sm text-slate-500">• {{ store.providerLabel(conn.provider) }}</span>
                  </div>
                </div>
              </div>

              <button *ngIf="canReconnect(conn.status)" class="btn-secondary inline-flex items-center gap-2" (click)="reconnect(conn.provider)" [disabled]="submitting()">
                <span *ngIf="submitting()" class="inline-block h-4 w-4 animate-spin rounded-full border-2 border-slate-300 border-t-slate-600"></span>
                <span *ngIf="!submitting()" class="material-icons-outlined text-[18px]">sync</span>
                Reconectar cuenta
              </button>
            </div>
          </div>

          <!-- Opciones de Compartir/Visibilidad -->
          <div class="card p-6 space-y-4">
            <div>
              <h3 class="text-base font-semibold text-slate-900">Visibilidad de la cuenta</h3>
              <p class="text-sm text-slate-500 mt-1">Controla quién puede ver los correos e interacciones de esta cuenta dentro de la organización.</p>
            </div>

            <div class="space-y-3">
              <label
                class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 cursor-pointer hover:bg-slate-50 transition-colors"
                [ngClass]="{ 'ring-2 ring-indigo-500 border-indigo-500 bg-indigo-50/30': conn.sharing_policy === 'private' }"
              >
                <div class="flex h-5 items-center">
                  <input
                    type="radio"
                    name="sharing_policy"
                    value="private"
                    [checked]="conn.sharing_policy === 'private'"
                    (change)="updateSharingPolicy('private')"
                    [disabled]="submitting()"
                    class="h-4 w-4 text-indigo-600 border-slate-300 focus:ring-indigo-600"
                  />
                </div>
                <div class="flex-1">
                  <div class="flex items-center justify-between">
                    <span class="block text-sm font-medium text-slate-900">Privado (Solo yo)</span>
                    <span class="material-icons-outlined text-slate-400 text-[18px]">lock</span>
                  </div>
                  <span class="block text-sm text-slate-500 mt-1">Solo tú podrás ver los correos sincronizados desde esta cuenta.</span>
                </div>
              </label>

              <label
                class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 cursor-pointer hover:bg-slate-50 transition-colors"
                [ngClass]="{ 'ring-2 ring-indigo-500 border-indigo-500 bg-indigo-50/30': conn.sharing_policy === 'tenant_all' }"
              >
                <div class="flex h-5 items-center">
                  <input
                    type="radio"
                    name="sharing_policy"
                    value="tenant_all"
                    [checked]="conn.sharing_policy === 'tenant_all'"
                    (change)="updateSharingPolicy('tenant_all')"
                    [disabled]="submitting()"
                    class="h-4 w-4 text-indigo-600 border-slate-300 focus:ring-indigo-600"
                  />
                </div>
                <div class="flex-1">
                  <div class="flex items-center justify-between">
                    <span class="block text-sm font-medium text-slate-900">Compartido (Equipo)</span>
                    <span class="material-icons-outlined text-slate-400 text-[18px]">group</span>
                  </div>
                  <span class="block text-sm text-slate-500 mt-1">Cualquier miembro de {{ tenantName()?.name || 'la organización' }} podrá ver los correos sincronizados.</span>
                </div>
              </label>
            </div>

            <div *ngIf="submitting()" class="flex items-center gap-2 text-sm text-indigo-600 mt-2">
              <span class="inline-block h-4 w-4 animate-spin rounded-full border-2 border-indigo-200 border-t-indigo-600"></span>
              Actualizando preferencias...
            </div>
          </div>

          <!-- Zona Peligrosa -->
          <div class="card p-6 border-rose-200 bg-rose-50/30">
            <div>
              <h3 class="text-base font-semibold text-rose-700">Zona Peligrosa</h3>
              <p class="text-sm text-rose-600 mt-1">Acciones irreversibles para esta conexión.</p>
            </div>

            <div class="mt-4 flex items-center justify-between border-t border-rose-100 pt-4">
              <div>
                <p class="text-sm font-medium text-slate-900">Desvincular cuenta</p>
                <p class="text-sm text-slate-500">Se detendrá la sincronización de correos inmediatamente.</p>
              </div>
              <button
                class="rounded bg-rose-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-rose-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-rose-600 disabled:opacity-50"
                (click)="openDisconnectModal()"
              >
                Desvincular
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal de confirmación para desvincular -->
    <app-modal [isOpen]="isDisconnectModalOpen" title="¿Estás seguro?" (close)="closeDisconnectModal()" [showDefaultFooter]="false">
      <div class="space-y-4">
        <div class="flex items-start gap-4">
          <div class="mx-auto flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-rose-100 sm:mx-0 sm:h-10 sm:w-10">
            <span class="material-icons-outlined text-rose-600">warning</span>
          </div>
          <div>
            <p class="text-sm text-slate-600 leading-relaxed">
              Estás a punto de desvincular la cuenta <span class="font-semibold text-slate-900">{{ connection()?.provider_account_email }}</span
              >.
            </p>
            <ul class="mt-3 text-sm text-slate-600 list-disc pl-5 space-y-1">
              <li>Se detendrá la sincronización de correos de forma inmediata.</li>
              <li>La cuenta no podrá ser utilizada para futuras automatizaciones.</li>
              <li>No se eliminarán los documentos o correos previamente sincronizados.</li>
            </ul>
            <p class="mt-4 text-sm font-medium text-slate-900">Esta acción no se puede deshacer.</p>
          </div>
        </div>
      </div>

      <ng-container modal-footer>
        <button
          type="button"
          class="w-full inline-flex justify-center rounded-md bg-rose-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-rose-500 sm:w-auto disabled:opacity-50"
          (click)="confirmDisconnect()"
          [disabled]="disconnectingId() === connection()?.id"
        >
          <span *ngIf="disconnectingId() === connection()?.id" class="inline-block h-4 w-4 animate-spin rounded-full border-2 border-white/20 border-t-white mr-2"></span>
          Sí, desvincular
        </button>
        <button
          type="button"
          class="mt-3 w-full inline-flex justify-center rounded-md bg-white px-3 py-2 text-sm font-semibold text-slate-900 shadow-sm ring-1 ring-inset ring-slate-300 hover:bg-slate-50 sm:mt-0 sm:w-auto"
          (click)="closeDisconnectModal()"
          [disabled]="disconnectingId() === connection()?.id"
        >
          Cancelar
        </button>
      </ng-container>
    </app-modal>
  `,
})
export class ConnectionDetailsComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly location = inject(Location);
  readonly store = inject(ConnectionsStore);
  private readonly tenantContextStore = inject(TenantContextStore);

  readonly connection = this.store.selectedConnection;
  readonly loading = this.store.loading;
  readonly submitting = this.store.submitting;
  readonly disconnectingId = this.store.disconnectingId;
  readonly tenantName = this.tenantContextStore.tenantDetails;

  isDisconnectModalOpen = false;
  private connectionId!: string;

  ngOnInit(): void {
    const tenantId = this.route.snapshot.paramMap.get('tenantId');
    if (tenantId) {
      this.tenantContextStore.setTenantId(tenantId);
    }

    this.connectionId = this.route.snapshot.paramMap.get('connectionId') || '';
    if (this.connectionId) {
      this.store.loadConnection(this.connectionId);
    }
  }

  goBack(): void {
    this.location.back();
  }

  updateSharingPolicy(policy: 'private' | 'tenant_all'): void {
    if (this.connection()?.sharing_policy === policy) return;
    this.store.updateSharingPolicy(this.connectionId, policy);
  }

  canReconnect(status: ConnectionStatus): boolean {
    return status === 'requires_reconnect';
  }

  reconnect(provider: ConnectionProvider): void {
    this.store.connectProvider(provider, (authURL) => window.location.assign(authURL));
  }

  openDisconnectModal(): void {
    this.isDisconnectModalOpen = true;
  }

  closeDisconnectModal(): void {
    if (this.disconnectingId() === this.connectionId) return;
    this.isDisconnectModalOpen = false;
  }

  confirmDisconnect(): void {
    if (!this.connection()) return;

    this.store.disconnectConnection(this.connectionId, () => {
      this.closeDisconnectModal();
      this.router.navigate(['/', this.tenantContextStore.tenantId(), 'connections']);
    });
  }
}
