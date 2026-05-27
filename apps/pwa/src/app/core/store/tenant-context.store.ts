import { Injectable, inject, signal } from '@angular/core';
import { OrganizationHttpService, OrganizationResponse } from '../../organization/infrastructure/organization.http.service';

@Injectable({ providedIn: 'root' })
export class TenantContextStore {
  readonly tenantId = signal('');
  readonly tenantDetails = signal<OrganizationResponse | null>(null);

  private organizationService = inject(OrganizationHttpService);

  setTenantId(id: string) {
    if (this.tenantId() !== id) {
      this.tenantId.set(id);
      this.fetchTenantDetails(id);
    }
  }

  private fetchTenantDetails(id: string) {
    this.organizationService.getOrganization(id).subscribe({
      next: (response) => {
        const data = (response as any).data ? (response as any).data : response;
        this.tenantDetails.set(data);
      },
      error: (err) => {
        console.error('Failed to load tenant details', err);
      },
    });
  }
}
