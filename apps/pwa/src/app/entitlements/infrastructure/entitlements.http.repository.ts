import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { PlatformTenant, SetAccessPayload, TenantEntitlements, TenantEntitlementsDetail } from '../domain/entitlements.model';

@Injectable({ providedIn: 'root' })
export class EntitlementsHttpRepository {
  private readonly http = inject(HttpClient);
  private readonly apiUrl = environment.apiUrl;

  getEntitlements(): Observable<TenantEntitlements> {
    return this.http.get<TenantEntitlements>(`${this.apiUrl}/api/v1/entitlements`);
  }

  listPlatformTenants(): Observable<{ data: PlatformTenant[] }> {
    return this.http.get<{ data: PlatformTenant[] }>(`${this.apiUrl}/api/v1/platform/tenants`);
  }

  getPlatformTenantEntitlements(tenantId: string): Observable<TenantEntitlementsDetail> {
    return this.http.get<TenantEntitlementsDetail>(`${this.apiUrl}/api/v1/platform/tenants/${encodeURIComponent(tenantId)}/entitlements`);
  }

  setPlatformTenantAccess(tenantId: string, payload: SetAccessPayload): Observable<TenantEntitlementsDetail> {
    return this.http.put<TenantEntitlementsDetail>(`${this.apiUrl}/api/v1/platform/tenants/${encodeURIComponent(tenantId)}/entitlements`, payload);
  }
}
