import { CommonModule, DatePipe } from '@angular/common';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { BrnDialogContent } from '@spartan-ng/brain/dialog';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmDialogImports } from '@spartan-ng/helm/dialog';
import { HlmEmptyImports } from '@spartan-ng/helm/empty';
import { HlmInputImports } from '@spartan-ng/helm/input';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { PermissionsStore } from '../../../../rbac/application/permissions.store';
import { SecretsStore } from '../../../application/secrets.store';
import { SECRET_PURPOSE_CATALOG, Secret, SecretPurpose, SecretPurposes, purposeDefinition, purposeLabel, purposeShortLabel } from '../../../domain/secrets.model';
import { TenantContextStore } from '../../../../core/store/tenant-context.store';

@Component({
  selector: 'app-secrets-master',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    DatePipe,
    NgIcon,
    HlmCardImports,
    HlmButtonImports,
    HlmDialogImports,
    HlmInputImports,
    HlmSpinnerImports,
    HlmEmptyImports,
    HlmAlertImports,
    HlmBadgeImports,
    BrnDialogContent,
  ],
  host: {
    class: 'flex-1 flex flex-col min-h-0 w-full',
  },
  template: `
    <div class="h-full w-full flex-1 overflow-y-auto p-8">
      <div class="mx-auto w-full max-w-4xl space-y-6">
        <header class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div class="min-w-0 space-y-2">
            <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">Credenciales</h1>
            <p class="max-w-2xl text-sm text-muted-foreground">
              Registra aquí las claves que Bowerbird necesita para trabajar en nombre de la organización: APIs de sistemas externos, identidad (IdP), contraseñas de documentos y otras credenciales. Se
              guardan cifradas; el valor nunca se vuelve a mostrar.
            </p>
          </div>
          @if (permissions.canWriteSecrets()) {
            <button hlmBtn class="shrink-0" (click)="openCreateModal()">
              <ng-icon name="lucidePlus" />
              Agregar credencial
            </button>
          }
        </header>

        <div class="grid gap-3 sm:grid-cols-2">
          @for (item of purposeCatalog; track item.purpose) {
            <div class="rounded-lg border bg-card/40 px-4 py-3">
              <div class="text-sm font-medium">{{ item.label }}</div>
              <p class="mt-1 text-xs leading-relaxed text-muted-foreground">{{ item.description }}</p>
            </div>
          }
        </div>

        @if (store.errorMessage()) {
          <div hlmAlert variant="destructive">
            <ng-icon name="lucideCircleAlert" hlmAlertIcon />
            <h4 hlmAlertTitle>Error</h4>
            <p hlmAlertDescription>{{ store.errorMessage() }}</p>
          </div>
        }

        <hlm-card class="overflow-hidden p-0">
          <div class="flex items-center justify-between border-b bg-muted/30 px-6 py-4">
            <div>
              <h2 class="text-sm font-semibold uppercase tracking-wide">Credenciales guardadas</h2>
              <p class="mt-1 text-xs text-muted-foreground">Solo metadatos visibles. El sistema usa estos valores en segundo plano cuando hace falta.</p>
            </div>
            @if (!store.loading() && store.secrets().length > 0) {
              <span hlmBadge variant="secondary">{{ store.secrets().length }}</span>
            }
          </div>

          @if (store.loading()) {
            <div class="px-6 py-12 text-center text-sm text-muted-foreground">
              <hlm-spinner class="mx-auto mb-2 size-6" />
              Cargando credenciales...
            </div>
          } @else if (store.secrets().length === 0) {
            <div class="px-6 py-12">
              <div hlmEmpty>
                <div hlmEmptyHeader>
                  <div hlmEmptyMedia variant="icon">
                    <ng-icon name="lucideKeyRound" />
                  </div>
                  <h3 hlmEmptyTitle>Aún no hay credenciales</h3>
                  <p hlmEmptyDescription>
                    Cuando integres un ERP, configures un IdP o necesites desbloquear documentos protegidos, agrega aquí la clave correspondiente para que el sistema pueda usarla de forma segura.
                  </p>
                </div>
              </div>
            </div>
          } @else {
            <ul class="divide-y">
              @for (secret of store.secrets(); track secret.id) {
                <li class="flex flex-col gap-3 px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
                  <div class="min-w-0 space-y-1.5">
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="truncate font-medium">{{ secret.label }}</span>
                      <span hlmBadge variant="outline">{{ purposeShortLabel(secret.purpose) }}</span>
                    </div>
                    <p class="text-xs text-muted-foreground">{{ purposeLabel(secret.purpose) }}</p>
                    <div class="text-xs text-muted-foreground">
                      Versión {{ secret.version }}
                      @if (secret.last_used_at) {
                        · Último uso {{ secret.last_used_at | date: 'short' }}
                      }
                      · Actualizado {{ secret.updated_at | date: 'short' }}
                    </div>
                    <div class="font-mono text-xs tracking-widest text-muted-foreground">••••••••••••</div>
                  </div>
                  <div class="flex shrink-0 gap-2">
                    @if (permissions.canWriteSecrets()) {
                      <button hlmBtn variant="outline" size="sm" (click)="openRotateModal(secret)">Rotar valor</button>
                    }
                    @if (permissions.canDeleteSecrets()) {
                      <button hlmBtn variant="destructive" size="sm" (click)="deleteSecret(secret)" title="Eliminar">
                        <ng-icon name="lucideTrash2" />
                      </button>
                    }
                  </div>
                </li>
              }
            </ul>
          }
        </hlm-card>
      </div>
    </div>

    <hlm-dialog [state]="createOpen() ? 'open' : 'closed'" (stateChanged)="onCreateState($event)">
      <hlm-dialog-content *brnDialogContent="let ctx" class="sm:max-w-lg">
        <hlm-dialog-header>
          <h3 hlmDialogTitle>Nueva credencial</h3>
          <p hlmDialogDescription>Elige el tipo de secreto y una etiqueta que tu equipo reconozca. El valor se cifra al guardar y no se podrá consultar después.</p>
        </hlm-dialog-header>
        <div class="space-y-4 py-2">
          <div class="space-y-1.5">
            <label class="text-sm font-medium" for="secret-purpose">Tipo de credencial</label>
            <select hlmInput id="secret-purpose" class="w-full" [ngModel]="newPurpose()" (ngModelChange)="newPurpose.set($event)" name="newPurpose">
              @for (item of purposeCatalog; track item.purpose) {
                <option [value]="item.purpose">{{ item.label }}</option>
              }
            </select>
            @if (selectedPurposeDef(); as def) {
              <p class="text-xs text-muted-foreground">{{ def.description }}</p>
            }
          </div>
          <div class="space-y-1.5">
            <label class="text-sm font-medium" for="secret-label">Nombre / etiqueta</label>
            <input hlmInput id="secret-label" class="w-full" [(ngModel)]="newLabel" name="newLabel" [placeholder]="selectedPurposeDef()?.labelPlaceholder || 'Nombre reconocible'" />
          </div>
          <div class="space-y-1.5">
            <label class="text-sm font-medium" for="secret-value">{{ selectedPurposeDef()?.valueLabel || 'Valor' }}</label>
            <input
              hlmInput
              id="secret-value"
              class="w-full"
              type="password"
              [(ngModel)]="newValue"
              name="newValue"
              [placeholder]="selectedPurposeDef()?.valuePlaceholder || '••••••••'"
              autocomplete="new-password"
            />
          </div>
        </div>
        <hlm-dialog-footer>
          <button hlmBtn variant="outline" (click)="createOpen.set(false)">Cancelar</button>
          <button hlmBtn [disabled]="store.submitting() || !newLabel.trim() || !newValue.trim()" (click)="submitCreate()">Guardar de forma segura</button>
        </hlm-dialog-footer>
      </hlm-dialog-content>
    </hlm-dialog>

    <hlm-dialog [state]="rotateOpen() ? 'open' : 'closed'" (stateChanged)="onRotateState($event)">
      <hlm-dialog-content *brnDialogContent="let ctx" class="sm:max-w-md">
        <hlm-dialog-header>
          <h3 hlmDialogTitle>Rotar valor</h3>
          <p hlmDialogDescription>Reemplaza el valor de «{{ rotateTarget()?.label }}». El valor anterior deja de usarse de inmediato.</p>
        </hlm-dialog-header>
        <div class="space-y-1.5 py-2">
          <label class="text-sm font-medium" for="rotate-value">Nuevo valor</label>
          <input hlmInput id="rotate-value" class="w-full" type="password" [(ngModel)]="rotateValue" name="rotateValue" placeholder="••••••••" autocomplete="new-password" />
        </div>
        <hlm-dialog-footer>
          <button hlmBtn variant="outline" (click)="rotateOpen.set(false)">Cancelar</button>
          <button hlmBtn [disabled]="store.submitting() || !rotateValue.trim()" (click)="submitRotate()">Actualizar</button>
        </hlm-dialog-footer>
      </hlm-dialog-content>
    </hlm-dialog>
  `,
})
export class MasterPage implements OnInit {
  readonly store = inject(SecretsStore);
  readonly permissions = inject(PermissionsStore);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly tenantContext = inject(TenantContextStore);

  readonly purposeCatalog = SECRET_PURPOSE_CATALOG;
  readonly purposeLabel = purposeLabel;
  readonly purposeShortLabel = purposeShortLabel;

  createOpen = signal(false);
  rotateOpen = signal(false);
  rotateTarget = signal<Secret | null>(null);
  newPurpose = signal<SecretPurpose>(SecretPurposes.IntegrationsApiKey);
  newLabel = '';
  newValue = '';
  rotateValue = '';

  readonly selectedPurposeDef = computed(() => purposeDefinition(this.newPurpose()));

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('tenantId');
    if (id) {
      this.tenantContext.setTenantId(id);
    }

    const tenantId = id || this.tenantContext.tenantId();
    this.permissions.load(tenantId).subscribe(() => {
      if (!this.permissions.canReadSecrets()) {
        void this.router.navigate(['/', tenantId, 'dashboard']);
        return;
      }
      this.store.loadSecrets();
    });
  }

  openCreateModal(): void {
    this.newPurpose.set(SecretPurposes.IntegrationsApiKey);
    this.newLabel = '';
    this.newValue = '';
    this.createOpen.set(true);
  }

  openRotateModal(secret: Secret): void {
    this.rotateTarget.set(secret);
    this.rotateValue = '';
    this.rotateOpen.set(true);
  }

  onCreateState(state: 'open' | 'closed'): void {
    this.createOpen.set(state === 'open');
  }

  onRotateState(state: 'open' | 'closed'): void {
    this.rotateOpen.set(state === 'open');
    if (state === 'closed') {
      this.rotateTarget.set(null);
    }
  }

  submitCreate(): void {
    this.store.createSecret({
      purpose: this.newPurpose(),
      label: this.newLabel.trim(),
      value: this.newValue.trim(),
    });
    this.createOpen.set(false);
  }

  submitRotate(): void {
    const target = this.rotateTarget();
    if (!target) return;
    this.store.rotateSecret(target.id, this.rotateValue.trim());
    this.rotateOpen.set(false);
  }

  deleteSecret(secret: Secret): void {
    if (!confirm(`¿Eliminar la credencial «${secret.label}»? El sistema dejará de poder usarla.`)) return;
    this.store.deleteSecret(secret.id);
  }
}
