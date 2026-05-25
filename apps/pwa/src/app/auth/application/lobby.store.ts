import { Injectable, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { AuthStore } from './auth.store';
import { TenantMembership } from '../domain/auth.model';
import { OrganizationHttpService } from '../../organization/infrastructure/organization.http.service';

@Injectable({ providedIn: 'root' })
export class LobbyStore {
  readonly showCreateForm = signal(false);
  readonly newOrgName = signal('');
  readonly newOrgSlug = signal('');
  readonly isCreating = signal(false);
  readonly createError = signal('');

  private readonly authStore = inject(AuthStore);
  private readonly orgService = inject(OrganizationHttpService);
  private readonly router = inject(Router);

  init(): void {
    if (!this.authStore.isAuthenticated()) {
      void this.router.navigate(['/login']);
      return;
    }

    this.authStore.loadTenants();
  }

  toggleCreateForm(): void {
    const next = !this.showCreateForm();
    this.showCreateForm.set(next);
    this.createError.set('');

    if (!next) {
      this.newOrgName.set('');
      this.newOrgSlug.set('');
    }
  }

  onNameChange(name: string): void {
    const currentSlug = this.newOrgSlug();
    if (!currentSlug || currentSlug === this.generateSlug(name.slice(0, -1))) {
      this.newOrgSlug.set(this.generateSlug(name));
    }
  }

  selectTenant(tenant: TenantMembership): void {
    void this.router.navigate(['/', tenant.tenant_id, 'dashboard']);
  }

  goToConnections(tenant: TenantMembership): void {
    void this.router.navigate(['/', tenant.tenant_id, 'inbox', 'connections']);
  }

  goToUnifiedInbox(tenant: TenantMembership): void {
    void this.router.navigate(['/', tenant.tenant_id, 'inbox', 'unified']);
  }

  createTenant(): void {
    const name = this.newOrgName();
    const slug = this.newOrgSlug();
    if (!name || !slug) {
      return;
    }

    this.isCreating.set(true);
    this.createError.set('');

    this.orgService.createOrganization({ name, slug }).subscribe({
      next: () => {
        this.isCreating.set(false);
        this.toggleCreateForm();
        this.authStore.loadTenants();
      },
      error: (err) => {
        this.isCreating.set(false);
        this.createError.set(err.error || 'Failed to create organization');
      },
    });
  }

  logout(): void {
    this.authStore.logout({
      onFinish: () => {
        void this.router.navigate(['/login']);
      },
    });
  }

  setNewOrgName(name: string): void {
    this.newOrgName.set(name);
  }

  setNewOrgSlug(slug: string): void {
    this.newOrgSlug.set(slug);
  }

  private generateSlug(text: string): string {
    return text
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/(^-|-$)+/g, '');
  }
}
