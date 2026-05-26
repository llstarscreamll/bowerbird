import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface CreateOrganizationRequest {
  name: string;
  slug: string;
}

export interface OrganizationResponse {
  id: string;
  name: string;
  slug: string;
  status: string;
  created_at: string;
  members_count?: number;
  current_user_role?: string;
}

@Injectable({ providedIn: 'root' })
export class OrganizationHttpService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/api/v1/organizations`;

  createOrganization(data: CreateOrganizationRequest): Observable<OrganizationResponse> {
    return this.http.post<OrganizationResponse>(this.baseUrl, data);
  }

  getOrganization(id: string): Observable<OrganizationResponse> {
    return this.http.get<OrganizationResponse>(`${this.baseUrl}/${id}`);
  }
}
