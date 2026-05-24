import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

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
}

@Injectable({ providedIn: 'root' })
export class OrganizationHttpService {
  private readonly http = inject(HttpClient);
  // In a real app, API_URL would come from environment
  private readonly baseUrl = 'http://api.bowerbird.dev/api/v1/organizations';

  createOrganization(data: CreateOrganizationRequest): Observable<OrganizationResponse> {
    return this.http.post<OrganizationResponse>(this.baseUrl, data);
  }
}
