import { Injectable, inject, signal } from '@angular/core';
import { TenantHttpService, TenantResponse } from '../../tenant/infrastructure/tenant.http.service';

@Injectable({ providedIn: 'root' })
export class TenantContextStore {
  readonly tenantId = signal('');
  readonly tenantDetails = signal<TenantResponse | null>(null);

  private tenantService = inject(TenantHttpService);

  setTenantId(id: string) {
    if (this.tenantId() !== id) {
      this.tenantId.set(id);
      this.fetchTenantDetails(id);
    }
  }

  private fetchTenantDetails(id: string) {
    this.tenantService.getTenant(id).subscribe({
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
