import { CommonModule } from '@angular/common';
import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { BrnAlertDialogContent } from '@spartan-ng/brain/alert-dialog';
import { BrnDialogContent } from '@spartan-ng/brain/dialog';
import { ConnectionsStore } from '../../../application/connections.store';
import { TenantContextStore } from '../../../../core/store/tenant-context.store';
import { Connection, ConnectionProvider, ConnectionStatus } from '../../../domain/connections.model';
import { IconGoogleComponent } from '../../../../core/presentation/components/icons/icon-google.component';
import { IconMicrosoftComponent } from '../../../../core/presentation/components/icons/icon-microsoft.component';
import { ConnectionStatusChipComponent } from '../../../../core/presentation/components/connection-status-chip/connection-status-chip.component';
import { HlmAlertDialogImports } from '@spartan-ng/helm/alert-dialog';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmDialogImports } from '@spartan-ng/helm/dialog';
import { HlmEmptyImports } from '@spartan-ng/helm/empty';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';

@Component({
  selector: 'app-connections-list',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterModule,
    NgIcon,
    ConnectionStatusChipComponent,
    IconGoogleComponent,
    IconMicrosoftComponent,
    HlmCardImports,
    HlmButtonImports,
    HlmDialogImports,
    HlmAlertDialogImports,
    HlmBadgeImports,
    HlmSpinnerImports,
    HlmEmptyImports,
    BrnDialogContent,
    BrnAlertDialogContent,
  ],
  host: {
    class: 'flex-1 flex flex-col min-h-0 w-full',
  },
  template: `
    <div class="h-full overflow-y-auto bg-gradient-to-b from-muted/30 to-background px-4 py-8 sm:px-6 lg:px-8">
      <div class="mx-auto w-full max-w-5xl space-y-8">
        <header class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">Conexiones</h1>
            <p class="mt-1 max-w-2xl text-sm text-muted-foreground">
              Gestiona las cuentas de correo vinculadas a la organización
              <span class="font-medium text-foreground">{{ tenantName()?.name || 'actual' }}</span
              >.
            </p>
          </div>
          <button hlmBtn (click)="openConnectModal()">
            <ng-icon name="lucidePlus" />
            Añadir cuenta
          </button>
        </header>

        <hlm-card class="overflow-hidden p-0">
          <div class="flex items-center justify-between border-b bg-muted/30 px-6 py-4">
            <h2 class="text-sm font-semibold uppercase tracking-wide">Cuentas vinculadas</h2>
            <div class="flex gap-2 text-xs">
              @if (statusCount('active') > 0) {
                <span hlmBadge variant="default">{{ statusCount('active') }} Activas</span>
              }
              @if (statusCount('requires_reconnect') > 0) {
                <span hlmBadge variant="secondary">{{ statusCount('requires_reconnect') }} Problemas</span>
              }
            </div>
          </div>

          @if (loading()) {
            <div class="px-6 py-12 text-center text-sm text-muted-foreground">
              <hlm-spinner class="mx-auto mb-2 size-6" />
              Cargando conexiones...
            </div>
          } @else if (connections().length === 0) {
            <hlm-empty class="py-16">
              <ng-icon hlm name="lucideUnlink" class="text-muted-foreground" />
              <h3 hlmEmptyTitle>No hay cuentas conectadas</h3>
              <p hlmEmptyDescription>Añade tu primera cuenta de correo para empezar a sincronizar facturas y comprobantes.</p>
              <button hlmBtn variant="outline" class="mt-4" (click)="openConnectModal()">Añadir primera cuenta</button>
            </hlm-empty>
          } @else {
            <ul class="divide-y">
              @for (conn of connections(); track conn.id) {
                <li class="group relative px-6 py-5 transition-colors hover:bg-muted/30">
                  <div class="flex items-start justify-between gap-4">
                    <div class="flex items-center gap-4">
                      <div class="flex size-10 shrink-0 items-center justify-center rounded-full border bg-card p-2.5 shadow-sm">
                        @if (conn.provider === 'gmail') {
                          <app-icon-google class="size-full object-contain" />
                        }
                        @if (conn.provider === 'microsoft') {
                          <app-icon-microsoft class="size-full object-contain" />
                        }
                      </div>
                      <div class="min-w-0">
                        <div class="flex items-center gap-2">
                          <p class="truncate text-sm font-semibold">{{ conn.provider_account_email }}</p>
                          <app-connection-status-chip [status]="conn.status" />
                        </div>
                        <div class="mt-1 flex items-center gap-3 text-xs text-muted-foreground">
                          <span>{{ providerLabel(conn.provider) }}</span>
                          <span class="size-1 rounded-full bg-muted-foreground/40"></span>
                          <span class="flex items-center gap-1">
                            <ng-icon name="lucideEye" class="text-sm" />
                            {{ conn.sharing_policy === 'private' ? 'Privado (Solo yo)' : 'Compartido (Equipo)' }}
                          </span>
                        </div>
                      </div>
                    </div>
                    <div class="flex items-center gap-2 opacity-0 transition-opacity group-hover:opacity-100">
                      <button hlmBtn variant="ghost" size="icon-sm" title="Detalles y configuración" [routerLink]="['/', tenantId(), 'connections', conn.id]">
                        <ng-icon name="lucideSettings" />
                      </button>
                      <button hlmBtn variant="ghost" size="icon-sm" title="Desvincular" (click)="openDisconnectConfirm(conn)">
                        <ng-icon name="lucideUnlink" />
                      </button>
                    </div>
                  </div>
                </li>
              }
            </ul>
          }
        </hlm-card>
      </div>
    </div>

    <hlm-dialog [state]="isConnectModalOpen ? 'open' : 'closed'" (closed)="closeConnectModal()">
      <hlm-dialog-content *brnDialogContent class="sm:max-w-lg">
        <hlm-dialog-header>
          <h2 hlmDialogTitle>Añadir nueva cuenta</h2>
          <p hlmDialogDescription>Selecciona tu proveedor de correo para vincular tu cuenta.</p>
        </hlm-dialog-header>

        <div class="space-y-6">
          <div class="rounded-lg border border-primary/20 bg-primary/5 p-4">
            <h4 class="flex items-center gap-2 text-sm font-semibold text-primary">
              <ng-icon name="lucideShield" />
              Permisos que solicitaremos
            </h4>
            <ul class="mt-2 list-inside list-disc space-y-1 text-sm text-muted-foreground">
              <li>Leer correos electrónicos (para encontrar facturas)</li>
              <li>Crear y asignar etiquetas (para organizar tu bandeja)</li>
            </ul>
          </div>

          <div class="grid gap-3">
            <button hlmBtn variant="outline" class="justify-center gap-3 py-3" (click)="connect('gmail')" [disabled]="submitting()">
              <app-icon-google class="size-5" />
              Continuar con Google
            </button>
            <button hlmBtn variant="outline" class="justify-center gap-3 py-3" disabled title="Próximamente">
              <app-icon-microsoft class="size-5" />
              Continuar con Microsoft (Pronto)
            </button>
          </div>
        </div>
      </hlm-dialog-content>
    </hlm-dialog>

    <hlm-alert-dialog [state]="disconnectTarget() ? 'open' : 'closed'" (closed)="closeDisconnectConfirm()">
      <hlm-alert-dialog-content *brnAlertDialogContent>
        <hlm-alert-dialog-header>
          <h2 hlmAlertDialogTitle>¿Desvincular cuenta?</h2>
          <p hlmAlertDialogDescription>Se desvinculará la cuenta {{ disconnectTarget()?.provider_account_email }}. ¿Deseas continuar?</p>
        </hlm-alert-dialog-header>
        <hlm-alert-dialog-footer>
          <button hlmAlertDialogCancel>Cancelar</button>
          <button hlmAlertDialogAction variant="destructive" (click)="confirmDisconnect()">Desvincular</button>
        </hlm-alert-dialog-footer>
      </hlm-alert-dialog-content>
    </hlm-alert-dialog>
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
  disconnectTarget = signal<Connection | null>(null);

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

  openDisconnectConfirm(conn: Connection): void {
    this.disconnectTarget.set(conn);
  }

  closeDisconnectConfirm(): void {
    this.disconnectTarget.set(null);
  }

  confirmDisconnect(): void {
    const conn = this.disconnectTarget();
    if (!conn) return;
    this.store.disconnectConnection(conn.id);
    this.closeDisconnectConfirm();
  }

  providerLabel(provider: ConnectionProvider): string {
    return this.store.providerLabel(provider);
  }

  statusCount(status: ConnectionStatus): number {
    return this.connections().filter((conn) => conn.status === status).length;
  }
}
