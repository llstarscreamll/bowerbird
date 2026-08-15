import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmCheckboxImports } from '@spartan-ng/helm/checkbox';
import { HlmInputImports } from '@spartan-ng/helm/input';
import { HlmLabelImports } from '@spartan-ng/helm/label';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { EntitlementsHttpRepository } from '../../../infrastructure/entitlements.http.repository';
import { EntitlementGrant, FeatureKeys, PlatformTenant, TenantEntitlementsDetail } from '../../../domain/entitlements.model';
import { ToastService } from '../../../../core/services/toast.service';

@Component({
  selector: 'app-platform-operators',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, HlmButtonImports, HlmCardImports, HlmCheckboxImports, HlmInputImports, HlmLabelImports, HlmSpinnerImports],
  template: `
    <div class="min-h-screen bg-muted/30 px-4 py-10 sm:px-6 lg:px-8">
      <div class="mx-auto max-w-5xl space-y-6">
        <header class="flex items-center justify-between border-b border-border pb-4">
          <div>
            <h1 class="text-2xl font-semibold tracking-tight">Panel de plataforma</h1>
            <p class="mt-1 text-sm text-muted-foreground">Accesos de producto por organización</p>
          </div>
          <a hlmBtn variant="outline" routerLink="/lobby">Volver al lobby</a>
        </header>

        <div class="grid gap-6 lg:grid-cols-[280px_1fr]">
          <hlm-card class="h-fit p-0">
            <div class="border-b border-border px-4 py-3 text-sm font-medium">Organizaciones</div>
            @if (loadingTenants()) {
              <div class="flex justify-center py-8">
                <hlm-spinner class="size-5" />
              </div>
            } @else {
              <ul class="divide-y divide-border">
                @for (tenant of tenants(); track tenant.id) {
                  <li>
                    <button type="button" class="w-full px-4 py-3 text-left text-sm hover:bg-muted/40" [class.bg-muted]="selectedTenantId() === tenant.id" (click)="selectTenant(tenant)">
                      <div class="font-medium">{{ tenant.name }}</div>
                      <div class="text-xs text-muted-foreground">{{ tenant.slug }}</div>
                    </button>
                  </li>
                }
              </ul>
            }
          </hlm-card>

          <hlm-card class="p-5">
            @if (!selectedTenant()) {
              <p class="text-sm text-muted-foreground">Selecciona una organización para gestionar sus accesos.</p>
            } @else if (loadingDetail()) {
              <div class="flex justify-center py-12">
                <hlm-spinner class="size-6" />
              </div>
            } @else {
              <div class="space-y-6">
                <div>
                  <h2 class="text-lg font-semibold">{{ selectedTenant()?.name }}</h2>
                  <p class="text-sm text-muted-foreground">{{ accessSummary() }}</p>
                </div>

                <div class="space-y-4">
                  <div class="rounded-lg border border-border p-4">
                    <div class="flex items-start justify-between gap-4">
                      <div>
                        <h3 class="font-medium">Facturas</h3>
                        <p class="mt-1 text-sm text-muted-foreground">Espacio de trabajo e ingesta desde correo.</p>
                      </div>
                      <span class="text-sm font-medium text-muted-foreground">{{ invoicingEnabled() ? 'Activo' : 'Apagado' }}</span>
                    </div>
                  </div>

                  <div class="rounded-lg border border-border p-4 space-y-3">
                    <div class="flex items-start justify-between gap-4">
                      <div>
                        <h3 class="font-medium">Correo</h3>
                        <p class="mt-1 text-sm text-muted-foreground">Leer y organizar el buzón conectado.</p>
                      </div>
                      <hlm-checkbox [checked]="mailEnabled()" (checkedChange)="onMailToggle($event)" aria-label="Activar correo" />
                    </div>
                    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
                      <label hlmLabel for="mailTrialUntil">Prueba hasta</label>
                      <input
                        hlmInput
                        id="mailTrialUntil"
                        type="date"
                        class="max-w-48"
                        [ngModel]="mailTrialUntil()"
                        (ngModelChange)="mailTrialUntil.set($event)"
                        [disabled]="!mailEnabled() || saving()"
                      />
                      <button type="button" hlmBtn size="sm" variant="outline" [disabled]="!mailEnabled() || saving()" (click)="saveMailTrial()">Guardar fecha</button>
                    </div>
                    <p class="text-xs text-muted-foreground">{{ mailStatusLabel() }}</p>
                  </div>

                  <div class="rounded-lg border border-border p-4">
                    <div class="flex items-start justify-between gap-4">
                      <div>
                        <h3 class="font-medium">Enviar</h3>
                        <p class="mt-1 text-sm text-muted-foreground">Redactar, responder y enviar en nombre de la cuenta.</p>
                      </div>
                      <hlm-checkbox [checked]="sendEnabled()" [disabled]="!mailEnabled()" (checkedChange)="onSendToggle($event)" aria-label="Activar enviar" />
                    </div>
                  </div>
                </div>
              </div>
            }
          </hlm-card>
        </div>
      </div>
    </div>
  `,
})
export class PlatformOperatorsPage implements OnInit {
  private readonly repo = inject(EntitlementsHttpRepository);
  private readonly toast = inject(ToastService);

  readonly tenants = signal<PlatformTenant[]>([]);
  readonly loadingTenants = signal(false);
  readonly loadingDetail = signal(false);
  readonly saving = signal(false);
  readonly selectedTenantId = signal('');
  readonly detail = signal<TenantEntitlementsDetail | null>(null);
  readonly mailTrialUntil = signal('');

  readonly selectedTenant = computed(() => this.tenants().find((tenant) => tenant.id === this.selectedTenantId()) ?? null);
  readonly mailEnabled = computed(() => this.detail()?.features.includes(FeatureKeys.MailInbox) ?? false);
  readonly sendEnabled = computed(() => this.detail()?.features.includes(FeatureKeys.MailSend) ?? false);
  readonly invoicingEnabled = computed(() => this.detail()?.features.includes(FeatureKeys.InvoicingWorkspace) ?? false);

  ngOnInit(): void {
    this.loadingTenants.set(true);
    this.repo.listPlatformTenants().subscribe({
      next: (response) => {
        this.tenants.set(response.data ?? []);
        this.loadingTenants.set(false);
      },
      error: () => {
        this.loadingTenants.set(false);
        this.toast.showError('No se pudieron cargar las organizaciones');
      },
    });
  }

  selectTenant(tenant: PlatformTenant): void {
    this.selectedTenantId.set(tenant.id);
    this.loadingDetail.set(true);
    this.repo.getPlatformTenantEntitlements(tenant.id).subscribe({
      next: (detail) => {
        this.detail.set(detail);
        this.mailTrialUntil.set(this.trialDateFromGrant(this.grantFor(detail.grants, FeatureKeys.MailInbox)));
        this.loadingDetail.set(false);
      },
      error: () => {
        this.loadingDetail.set(false);
        this.toast.showError('No se pudieron cargar los accesos');
      },
    });
  }

  accessSummary(): string {
    if (this.mailEnabled() && this.sendEnabled()) {
      return 'Cliente de correo completo';
    }
    if (this.mailEnabled()) {
      return 'Correo activo, enviar apagado';
    }
    return 'Correo apagado';
  }

  mailStatusLabel(): string {
    const grant = this.grantFor(this.detail()?.grants ?? [], FeatureKeys.MailInbox);
    if (!this.mailEnabled()) {
      return grant?.ends_at ? 'Expirado' : 'Apagado';
    }
    if (grant?.status === 'trial' && grant.ends_at) {
      return `Prueba hasta ${grant.ends_at.slice(0, 10)}`;
    }
    return 'Activo';
  }

  onMailToggle(enabled: boolean): void {
    this.saveAccess({ product: 'mail', enabled, ends_at: this.endsAtPayload() });
  }

  onSendToggle(enabled: boolean): void {
    this.saveAccess({ feature: FeatureKeys.MailSend, enabled });
  }

  saveMailTrial(): void {
    if (!this.mailEnabled()) {
      return;
    }
    this.saveAccess({ product: 'mail', enabled: true, ends_at: this.endsAtPayload() });
  }

  private saveAccess(payload: { product?: string; feature?: string; enabled: boolean; ends_at?: string | null }): void {
    const tenantId = this.selectedTenantId();
    if (!tenantId) {
      return;
    }
    this.saving.set(true);
    this.repo.setPlatformTenantAccess(tenantId, payload).subscribe({
      next: (detail) => {
        this.detail.set(detail);
        this.mailTrialUntil.set(this.trialDateFromGrant(this.grantFor(detail.grants, FeatureKeys.MailInbox)));
        this.saving.set(false);
        this.toast.showSuccess('Acceso actualizado');
      },
      error: () => {
        this.saving.set(false);
        this.toast.showError('No se pudo actualizar el acceso');
      },
    });
  }

  private endsAtPayload(): string | null {
    const value = this.mailTrialUntil();
    if (!value) {
      return null;
    }
    return `${value}T23:59:59Z`;
  }

  private grantFor(grants: EntitlementGrant[], featureKey: string): EntitlementGrant | undefined {
    return grants.find((grant) => grant.feature_key === featureKey);
  }

  private trialDateFromGrant(grant: EntitlementGrant | undefined): string {
    return grant?.ends_at ? grant.ends_at.slice(0, 10) : '';
  }
}
