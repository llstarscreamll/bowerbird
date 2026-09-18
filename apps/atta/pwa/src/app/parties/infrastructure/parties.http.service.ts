import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { CreatePartyInput, Party, UpdatePartyInput } from '../domain/party.model';

type PartyDoc = { data: { id: string; attributes: Omit<Party, 'id'> } };

const emptyParty = (attrs: Partial<Omit<Party, 'id'>>): Omit<Party, 'id'> => ({
  tax_id: attrs.tax_id ?? '',
  scheme_id: attrs.scheme_id ?? '',
  taxpayer_kind: attrs.taxpayer_kind ?? '',
  tax_level_codes: attrs.tax_level_codes ?? [],
  name: attrs.name ?? '',
  roles: attrs.roles ?? [],
  status: attrs.status ?? '',
  creation_source: attrs.creation_source ?? '',
  emails: attrs.emails ?? [],
  phones: attrs.phones ?? [],
  addresses: attrs.addresses ?? [],
  created_at: attrs.created_at ?? '',
  updated_at: attrs.updated_at ?? '',
});

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
      .pipe(map((res) => res.data.map((doc) => ({ id: doc.id, ...emptyParty(doc.attributes) }))));
  }

  getParty(id: string): Observable<Party> {
    return this.http.get<PartyDoc>(`${this.apiDomain}/api/v1/parties/${id}`).pipe(map((res) => ({ id: res.data.id, ...emptyParty(res.data.attributes) })));
  }

  createParty(input: CreatePartyInput): Observable<Party> {
    return this.http
      .post<PartyDoc>(`${this.apiDomain}/api/v1/parties`, {
        data: {
          type: 'parties',
          attributes: {
            name: input.name,
            tax_id: input.tax_id,
            scheme_id: input.scheme_id,
            roles: input.roles,
          },
        },
      })
      .pipe(map((res) => ({ id: res.data.id, ...emptyParty(res.data.attributes) })));
  }

  updateParty(id: string, input: UpdatePartyInput): Observable<Party> {
    return this.http
      .patch<PartyDoc>(`${this.apiDomain}/api/v1/parties/${id}`, {
        data: { attributes: input },
      })
      .pipe(map((res) => ({ id: res.data.id, ...emptyParty(res.data.attributes) })));
  }

  addEmail(id: string, value: string): Observable<Party> {
    return this.http
      .post<PartyDoc>(`${this.apiDomain}/api/v1/parties/${id}/emails`, {
        data: { attributes: { value } },
      })
      .pipe(map((res) => ({ id: res.data.id, ...emptyParty(res.data.attributes) })));
  }

  addPhone(id: string, value: string): Observable<Party> {
    return this.http
      .post<PartyDoc>(`${this.apiDomain}/api/v1/parties/${id}/phones`, {
        data: { attributes: { value } },
      })
      .pipe(map((res) => ({ id: res.data.id, ...emptyParty(res.data.attributes) })));
  }

  addAddress(id: string, input: { line: string; city: string; department: string; postal_zone: string; country_code: string; kind: string }): Observable<Party> {
    return this.http
      .post<PartyDoc>(`${this.apiDomain}/api/v1/parties/${id}/addresses`, {
        data: { attributes: input },
      })
      .pipe(map((res) => ({ id: res.data.id, ...emptyParty(res.data.attributes) })));
  }

  deleteEmail(id: string, emailId: string): Observable<Party> {
    return this.http.delete<PartyDoc>(`${this.apiDomain}/api/v1/parties/${id}/emails/${emailId}`).pipe(map((res) => ({ id: res.data.id, ...emptyParty(res.data.attributes) })));
  }

  deletePhone(id: string, phoneId: string): Observable<Party> {
    return this.http.delete<PartyDoc>(`${this.apiDomain}/api/v1/parties/${id}/phones/${phoneId}`).pipe(map((res) => ({ id: res.data.id, ...emptyParty(res.data.attributes) })));
  }

  deleteAddress(id: string, addressId: string): Observable<Party> {
    return this.http.delete<PartyDoc>(`${this.apiDomain}/api/v1/parties/${id}/addresses/${addressId}`).pipe(map((res) => ({ id: res.data.id, ...emptyParty(res.data.attributes) })));
  }
}
