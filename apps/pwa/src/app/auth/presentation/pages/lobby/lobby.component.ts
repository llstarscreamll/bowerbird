import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { NgIcon } from '@ng-icons/core';
import { AuthStore } from '../../../application/auth.store';
import { LobbyStore } from '../../../application/lobby.store';
import { TenantMembership } from '../../../domain/auth.model';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmEmptyImports } from '@spartan-ng/helm/empty';
import { HlmFieldImports } from '@spartan-ng/helm/field';
import { HlmInputImports } from '@spartan-ng/helm/input';
import { HlmLabelImports } from '@spartan-ng/helm/label';
import { HlmSeparatorImports } from '@spartan-ng/helm/separator';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';

@Component({
  selector: 'app-lobby',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    NgIcon,
    HlmCardImports,
    HlmButtonImports,
    HlmFieldImports,
    HlmInputImports,
    HlmLabelImports,
    HlmAlertImports,
    HlmBadgeImports,
    HlmSpinnerImports,
    HlmEmptyImports,
    HlmSeparatorImports,
  ],
  template: `
    <div class="min-h-screen bg-muted/30 px-4 py-10 sm:px-6 lg:px-8">
      <div class="mx-auto max-w-5xl space-y-6">
        <header class="flex items-center justify-between border-b border-border pb-4">
          <div>
            <h1 class="text-2xl font-semibold tracking-tight">Bienvenido</h1>
            <p class="mt-1 text-sm text-muted-foreground">Selecciona una organización para continuar</p>
          </div>
          <button hlmBtn variant="outline" (click)="logout()">
            <ng-icon name="lucideLogOut" />
            <span class="hidden sm:inline">Cerrar sesión</span>
          </button>
        </header>

        <hlm-card class="overflow-hidden p-0">
          <div class="flex items-center justify-between border-b border-border px-5 py-4">
            <h3 class="text-sm font-medium">Tus Organizaciones</h3>
            <button hlmBtn size="sm" (click)="toggleCreateForm()">
              <ng-icon [name]="showCreateForm() ? 'lucideX' : 'lucidePlus'" />
              {{ showCreateForm() ? 'Cancelar' : 'Crear nueva' }}
            </button>
          </div>

          @if (showCreateForm()) {
            <div class="border-b border-border bg-muted/20 p-5">
              <form (ngSubmit)="onCreateTenant()" class="space-y-4">
                <div class="grid grid-cols-1 gap-5 md:grid-cols-2">
                  <hlm-field>
                    <label hlmLabel for="orgName">Nombre de la Organización</label>
                    <input hlmInput id="orgName" type="text" required [ngModel]="newOrgName()" (ngModelChange)="onOrgNameInput($event)" name="orgName" placeholder="Acme Corp" />
                  </hlm-field>
                  <hlm-field>
                    <label hlmLabel for="orgSlug">URL del espacio / Slug</label>
                    <input hlmInput id="orgSlug" type="text" required [ngModel]="newOrgSlug()" (ngModelChange)="setNewOrgSlug($event)" name="orgSlug" placeholder="acme" />
                  </hlm-field>
                </div>

                @if (createError()) {
                  <hlm-alert variant="destructive">
                    <ng-icon hlm name="lucideCircleAlert" />
                    <p hlmAlertDescription>{{ createError() }}</p>
                  </hlm-alert>
                }

                <div class="flex justify-end pt-2">
                  <button type="submit" hlmBtn [disabled]="isCreating()">
                    @if (isCreating()) {
                      <hlm-spinner class="size-4" />
                      Creando...
                    } @else {
                      Crear Organización
                    }
                  </button>
                </div>
              </form>
            </div>
          }

          @if (store.isLoading()) {
            <div class="flex flex-col items-center justify-center space-y-3 py-12">
              <hlm-spinner class="size-6 text-primary" />
              <span class="text-sm text-muted-foreground">Cargando organizaciones...</span>
            </div>
          } @else if (store.tenants().length === 0) {
            <hlm-empty class="py-16">
              <ng-icon hlm name="lucideBuilding2" class="text-muted-foreground" />
              <h3 hlmEmptyTitle>No se encontraron organizaciones</h3>
              <p hlmEmptyDescription>Aún no perteneces a ninguna organización. Crea una nueva para comenzar.</p>
            </hlm-empty>
          } @else {
            <ul class="divide-y divide-border">
              @for (tenant of store.tenants(); track tenant.tenant_id) {
                <li class="group cursor-pointer transition-colors hover:bg-muted/40" (click)="selectTenant(tenant)">
                  <div class="flex items-center justify-between px-5 py-4">
                    <div class="flex items-center gap-3">
                      <div class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/20 bg-primary/10 text-xs font-medium uppercase text-primary">
                        {{ (tenant.name || tenant.tenant_id).substring(0, 2) }}
                      </div>
                      <div class="text-sm font-medium group-hover:text-primary">{{ tenant.name || tenant.tenant_id }}</div>
                    </div>
                    <div class="flex items-center gap-4">
                      <span hlmBadge [variant]="tenant.role === 'OWNER' ? 'default' : 'secondary'">{{ tenant.role | titlecase }}</span>
                      <ng-icon name="lucideChevronRight" class="text-muted-foreground group-hover:text-primary" />
                    </div>
                  </div>
                </li>
              }
            </ul>
          }
        </hlm-card>
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
