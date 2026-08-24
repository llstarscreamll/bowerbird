import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { Party } from '../domain/party.model';

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
}
