import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { CatalogItem, CreateCatalogItemInput, UpdateCatalogItemInput } from '../domain/catalog.model';

@Injectable({ providedIn: 'root' })
export class CatalogHttpService {
  private readonly http = inject(HttpClient);
  private readonly apiDomain = environment.apiUrl;

  listItems(kind?: string, status?: string, search?: string): Observable<CatalogItem[]> {
    let params = new HttpParams();
    if (kind) params = params.set('kind', kind);
    if (status) params = params.set('status', status);
    if (search?.trim()) params = params.set('search', search.trim());
    return this.http
      .get<{ data: { id: string; attributes: Omit<CatalogItem, 'id'> }[] }>(`${this.apiDomain}/api/v1/catalog/items`, { params })
      .pipe(map((res) => res.data.map((doc) => ({ id: doc.id, ...doc.attributes }))));
  }

  getItem(id: string): Observable<CatalogItem> {
    return this.http
      .get<{ data: { id: string; attributes: Omit<CatalogItem, 'id'> } }>(`${this.apiDomain}/api/v1/catalog/items/${id}`)
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  createItem(input: CreateCatalogItemInput): Observable<CatalogItem> {
    return this.http
      .post<{ data: { id: string; attributes: Omit<CatalogItem, 'id'> } }>(`${this.apiDomain}/api/v1/catalog/items`, {
        data: {
          type: 'catalog_items',
          id: input.id,
          attributes: {
            name: input.name,
            kind: input.kind,
            internal_sku: input.internal_sku,
          },
        },
      })
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  updateItem(id: string, input: UpdateCatalogItemInput): Observable<CatalogItem> {
    return this.http
      .patch<{ data: { id: string; attributes: Omit<CatalogItem, 'id'> } }>(`${this.apiDomain}/api/v1/catalog/items/${id}`, {
        data: { attributes: input },
      })
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }
}
