import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { CatalogItem, CatalogReviewLine, LineDecisionPayload } from '../domain/catalog.model';

@Injectable({ providedIn: 'root' })
export class CatalogHttpService {
  private readonly http = inject(HttpClient);
  private readonly apiDomain = environment.apiUrl;

  listItems(kind?: string, status?: string): Observable<CatalogItem[]> {
    let params = new HttpParams();
    if (kind) params = params.set('kind', kind);
    if (status) params = params.set('status', status);
    return this.http
      .get<{ data: { id: string; attributes: Omit<CatalogItem, 'id'> }[] }>(`${this.apiDomain}/api/v1/catalog/items`, { params })
      .pipe(map((res) => res.data.map((doc) => ({ id: doc.id, ...doc.attributes }))));
  }

  listReviewQueue(): Observable<CatalogReviewLine[]> {
    return this.http.get<{ data: { id: string; attributes: Omit<CatalogReviewLine, 'id'> }[] }>(`${this.apiDomain}/api/v1/catalog/review-queue`).pipe(
      map((res) =>
        res.data.map((doc) => ({
          id: doc.id,
          ...doc.attributes,
        })),
      ),
    );
  }

  rememberDecision(lineId: string, payload: LineDecisionPayload): Observable<void> {
    return this.http.post<void>(`${this.apiDomain}/api/v1/catalog/lines/${lineId}/decisions`, {
      data: { attributes: payload },
    });
  }
}
