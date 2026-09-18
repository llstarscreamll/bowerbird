import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface CreateTenantRequest {
  name: string;
  slug: string;
}

export interface TenantResponse {
  id: string;
  name: string;
  slug: string;
  status: string;
  created_at: string;
  members_count?: number;
  current_user_role?: string;
}

@Injectable({ providedIn: 'root' })
export class TenantHttpService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/api/v1/tenants`;

  createTenant(data: CreateTenantRequest): Observable<TenantResponse> {
    return this.http.post<TenantResponse>(this.baseUrl, data);
  }

  getTenant(id: string): Observable<TenantResponse> {
    return this.http.get<TenantResponse>(`${this.baseUrl}/${id}`);
  }
}
