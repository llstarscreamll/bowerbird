import { CommonModule, Location } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { BrnAlertDialogContent } from '@spartan-ng/brain/alert-dialog';
import { BrnDialogClose } from '@spartan-ng/brain/dialog';
import { ConnectionsStore } from '../../../application/connections.store';
import { ConnectionProvider, ConnectionStatus } from '../../../domain/connections.model';
import { TenantContextStore } from '../../../../core/store/tenant-context.store';
import { IconGoogleComponent } from '../../../../core/presentation/components/icons/icon-google.component';
import { IconMicrosoftComponent } from '../../../../core/presentation/components/icons/icon-microsoft.component';
import { ConnectionStatusChipComponent } from '../../../../core/presentation/components/connection-status-chip/connection-status-chip.component';
import { HlmAlertDialogImports } from '@spartan-ng/helm/alert-dialog';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';

@Component({
  selector: 'app-connection-details',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterModule,
    NgIcon,
    IconGoogleComponent,
    IconMicrosoftComponent,
    ConnectionStatusChipComponent,
    HlmCardImports,
    HlmButtonImports,
    HlmAlertDialogImports,
    HlmSpinnerImports,
    BrnAlertDialogContent,
    BrnDialogClose,
  ],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full' },
  template: `
    <div class="h-full overflow-y-auto bg-gradient-to-b from-muted/30 to-background px-4 py-8 sm:px-6 lg:px-8">
      <div class="mx-auto w-full max-w-3xl space-y-8">
        <header class="flex items-center gap-4">
          <button hlmBtn variant="ghost" size="icon" (click)="goBack()">
            <ng-icon name="lucideArrowLeft" />
          </button>
          <div>
            <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">Detalles de la Conexión</h1>
            <p class="mt-1 text-sm text-muted-foreground">Configura las preferencias y visibilidad para esta cuenta.</p>
          </div>
        </header>

        @if (loading() && !connection()) {
          <hlm-card class="px-6 py-12 text-center text-sm text-muted-foreground">
            <hlm-spinner class="mx-auto mb-2 size-6" />
            Cargando detalles...
          </hlm-card>
        }

        @if (connection(); as conn) {
          <hlm-card class="p-6">
            <div class="flex items-start justify-between gap-4">
              <div class="flex items-center gap-4">
                <div class="flex size-16 shrink-0 items-center justify-center rounded-full border bg-card p-3 shadow-sm">
                  @if (conn.provider === 'gmail') {
                    <app-icon-google class="size-full object-contain" />
                  }
                  @if (conn.provider === 'microsoft') {
                    <app-icon-microsoft class="size-full object-contain" />
                  }
                </div>
                <div>
                  <h2 class="text-xl font-semibold">{{ conn.provider_account_email }}</h2>
                  <div class="mt-2 flex items-center gap-2">
                    <app-connection-status-chip [status]="conn.status" />
                    <span class="text-sm text-muted-foreground">• {{ store.providerLabel(conn.provider) }}</span>
                  </div>
                </div>
              </div>
              @if (canReconnect(conn.status)) {
                <button hlmBtn variant="outline" (click)="reconnect(conn.provider)" [disabled]="submitting()">
                  @if (submitting()) {
                    <hlm-spinner class="size-4" />
                  } @else {
                    <ng-icon name="lucideRefreshCw" />
                  }
                  Reconectar cuenta
                </button>
              }
            </div>
          </hlm-card>

          <hlm-card class="space-y-4 p-6">
            <div>
              <h3 class="text-base font-semibold">Visibilidad de la cuenta</h3>
              <p class="mt-1 text-sm text-muted-foreground">Controla quién puede ver los correos e interacciones de esta cuenta dentro de la organización.</p>
            </div>
            <div class="space-y-3">
              <label
                class="flex cursor-pointer items-start gap-3 rounded-lg border p-4 transition-colors hover:bg-muted/30"
                [class.ring-2]="conn.sharing_policy === 'private'"
                [class.ring-primary]="conn.sharing_policy === 'private'"
              >
                <input
                  type="radio"
                  name="sharing_policy"
                  value="private"
                  [checked]="conn.sharing_policy === 'private'"
                  (change)="updateSharingPolicy('private')"
                  [disabled]="submitting()"
                  class="mt-1"
                />
                <div class="flex-1">
                  <div class="flex items-center justify-between">
                    <span class="text-sm font-medium">Privado (Solo yo)</span>
                    <ng-icon name="lucideLock" class="text-muted-foreground" />
                  </div>
                  <span class="mt-1 block text-sm text-muted-foreground">Solo tú podrás ver los correos sincronizados desde esta cuenta.</span>
                </div>
              </label>
              <label
                class="flex cursor-pointer items-start gap-3 rounded-lg border p-4 transition-colors hover:bg-muted/30"
                [class.ring-2]="conn.sharing_policy === 'tenant_all'"
                [class.ring-primary]="conn.sharing_policy === 'tenant_all'"
              >
                <input
                  type="radio"
                  name="sharing_policy"
                  value="tenant_all"
                  [checked]="conn.sharing_policy === 'tenant_all'"
                  (change)="updateSharingPolicy('tenant_all')"
                  [disabled]="submitting()"
                  class="mt-1"
                />
                <div class="flex-1">
                  <div class="flex items-center justify-between">
                    <span class="text-sm font-medium">Compartido (Equipo)</span>
                    <ng-icon name="lucideUsers" class="text-muted-foreground" />
                  </div>
                  <span class="mt-1 block text-sm text-muted-foreground">Cualquier miembro de {{ tenantName()?.name || 'la organización' }} podrá ver los correos sincronizados.</span>
                </div>
              </label>
            </div>
            @if (submitting()) {
              <div class="flex items-center gap-2 text-sm text-primary">
                <hlm-spinner class="size-4" />
                Actualizando preferencias...
              </div>
            }
          </hlm-card>

          <hlm-card class="border-destructive/30 bg-destructive/5 p-6">
            <h3 class="text-base font-semibold text-destructive">Zona Peligrosa</h3>
            <p class="mt-1 text-sm text-destructive/80">Acciones irreversibles para esta conexión.</p>
            <div class="mt-4 flex items-center justify-between border-t border-destructive/20 pt-4">
              <div>
                <p class="text-sm font-medium">Desvincular cuenta</p>
                <p class="text-sm text-muted-foreground">Se detendrá la sincronización de correos inmediatamente.</p>
              </div>
              <button hlmBtn variant="destructive" (click)="openDisconnectModal()">Desvincular</button>
            </div>
          </hlm-card>
        }
      </div>
    </div>

    <hlm-alert-dialog [state]="isDisconnectModalOpen ? 'open' : 'closed'" (closed)="closeDisconnectModal()">
      <hlm-alert-dialog-content *brnAlertDialogContent>
        <hlm-alert-dialog-header>
          <h2 hlmAlertDialogTitle>¿Estás seguro?</h2>
          <p hlmAlertDialogDescription>Estás a punto de desvincular la cuenta {{ connection()?.provider_account_email }}.</p>
        </hlm-alert-dialog-header>
        <ul class="list-disc space-y-1 ps-5 text-sm text-muted-foreground">
          <li>Se detendrá la sincronización de correos de forma inmediata.</li>
          <li>La cuenta no podrá ser utilizada para futuras automatizaciones.</li>
          <li>No se eliminarán los documentos o correos previamente sincronizados.</li>
        </ul>
        <p class="text-sm font-medium">Esta acción no se puede deshacer.</p>
        <hlm-alert-dialog-footer>
          <button hlmAlertDialogCancel>Cancelar</button>
          <button hlmAlertDialogAction variant="destructive" (click)="confirmDisconnect()" [disabled]="disconnectingId() === connection()?.id">
            @if (disconnectingId() === connection()?.id) {
              <hlm-spinner class="size-4" />
            }
            Sí, desvincular
          </button>
        </hlm-alert-dialog-footer>
      </hlm-alert-dialog-content>
    </hlm-alert-dialog>
  `,
})
export class ConnectionDetailsPage implements OnInit {
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
