import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { environment } from '../../../environments/environment';
import { Secret, SecretCollectionResponse, SecretDocumentResponse, mapSecret } from '../domain/secrets.model';

@Injectable({ providedIn: 'root' })
export class SecretsHttpRepository {
  private readonly http = inject(HttpClient);
  private readonly apiUrl = environment.apiUrl;

  list(purpose?: string): Observable<Secret[]> {
    let params = new HttpParams();
    if (purpose) {
      params = params.set('purpose', purpose);
    }
    return this.http.get<SecretCollectionResponse>(`${this.apiUrl}/api/v1/secrets`, { params }).pipe(map((response) => (response.data ?? []).map(mapSecret)));
  }

  create(input: { purpose: string; label: string; value: string; description?: string }): Observable<Secret> {
    return this.http
      .post<SecretDocumentResponse>(`${this.apiUrl}/api/v1/secrets`, {
        data: {
          type: 'secrets',
          attributes: input,
        },
      })
      .pipe(map((response) => mapSecret(response.data)));
  }

  update(id: string, input: { label?: string; value?: string; description?: string }): Observable<Secret> {
    return this.http
      .put<SecretDocumentResponse>(`${this.apiUrl}/api/v1/secrets/${encodeURIComponent(id)}`, {
        data: {
          type: 'secrets',
          attributes: input,
        },
      })
      .pipe(map((response) => mapSecret(response.data)));
  }

  delete(id: string): Observable<void> {
    return this.http.delete(`${this.apiUrl}/api/v1/secrets/${encodeURIComponent(id)}`, { responseType: 'text' }).pipe(map(() => void 0));
  }
}
