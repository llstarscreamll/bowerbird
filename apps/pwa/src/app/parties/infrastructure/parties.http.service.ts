import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { CreatePartyInput, Party, UpdatePartyInput } from '../domain/party.model';

@Injectable({ providedIn: 'root' })
export class PartiesHttpService {
  private readonly http = inject(HttpClient);
  private readonly apiDomain = environment.apiUrl;

  list(role?: string, search?: string): Observable<Party[]> {
    let params = new HttpParams();
    if (role) params = params.set('role', role);
    if (search) params = params.set('search', search);
    return this.http
      .get<{ data: { id: string; attributes: Omit<Party, 'id'> }[] }>(`${this.apiDomain}/api/v1/parties`, { params })
      .pipe(map((res) => res.data.map((doc) => ({ id: doc.id, ...doc.attributes }))));
  }

  getParty(id: string): Observable<Party> {
    return this.http.get<{ data: { id: string; attributes: Omit<Party, 'id'> } }>(`${this.apiDomain}/api/v1/parties/${id}`).pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  createParty(input: CreatePartyInput): Observable<Party> {
    return this.http
      .post<{ data: { id: string; attributes: Omit<Party, 'id'> } }>(`${this.apiDomain}/api/v1/parties`, {
        data: {
          type: 'parties',
          attributes: {
            name: input.name,
            tax_id: input.tax_id,
            roles: input.roles,
          },
        },
      })
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  updateParty(id: string, input: UpdatePartyInput): Observable<Party> {
    return this.http
      .patch<{ data: { id: string; attributes: Omit<Party, 'id'> } }>(`${this.apiDomain}/api/v1/parties/${id}`, {
        data: { attributes: input },
      })
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }
}
