import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { CatalogAlias, CatalogItem, CreateCatalogAliasInput, CreateCatalogItemInput, DuplicateCluster, DuplicateClusterMember, MergeItemsInput, UpdateCatalogItemInput } from '../domain/catalog.model';
import { CatalogImport, CatalogImportError, CatalogPage } from '../domain/catalog-import.model';

type JsonApiDoc<T> = { id: string; attributes: T };
type JsonApiList<T> = { data: JsonApiDoc<T>[]; meta?: { has_more?: boolean; cursor?: string; total?: number } };

@Injectable({ providedIn: 'root' })
export class CatalogHttpService {
  private readonly http = inject(HttpClient);
  private readonly apiDomain = environment.apiUrl;

  listItems(opts: { kind?: string; status?: string; search?: string; pageSize?: number; after?: string } = {}): Observable<CatalogPage<CatalogItem>> {
    let params = new HttpParams();
    if (opts.kind) params = params.set('kind', opts.kind);
    if (opts.status) params = params.set('status', opts.status);
    if (opts.search?.trim()) params = params.set('search', opts.search.trim());
    if (opts.pageSize) params = params.set('page[size]', String(opts.pageSize));
    if (opts.after) params = params.set('page[after]', opts.after);
    return this.http.get<JsonApiList<Omit<CatalogItem, 'id'>>>(`${this.apiDomain}/api/v1/catalog/items`, { params }).pipe(
      map((res) => ({
        items: res.data.map((doc) => ({ id: doc.id, ...doc.attributes })),
        hasMore: Boolean(res.meta?.has_more),
        cursor: res.meta?.cursor,
      })),
    );
  }

  getItem(id: string): Observable<CatalogItem> {
    return this.http.get<{ data: JsonApiDoc<Omit<CatalogItem, 'id'>> }>(`${this.apiDomain}/api/v1/catalog/items/${id}`).pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  createItem(input: CreateCatalogItemInput): Observable<CatalogItem> {
    return this.http
      .post<{ data: JsonApiDoc<Omit<CatalogItem, 'id'>> }>(`${this.apiDomain}/api/v1/catalog/items`, {
        data: {
          type: 'catalog_items',
          id: input.id,
          attributes: {
            name: input.name,
            kind: input.kind,
            internal_code: input.internal_code,
          },
        },
      })
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  updateItem(id: string, input: UpdateCatalogItemInput): Observable<CatalogItem> {
    return this.http
      .patch<{ data: JsonApiDoc<Omit<CatalogItem, 'id'>> }>(`${this.apiDomain}/api/v1/catalog/items/${id}`, {
        data: { attributes: input },
      })
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  addAlias(itemId: string, input: CreateCatalogAliasInput): Observable<CatalogAlias> {
    return this.http
      .post<{ data: JsonApiDoc<Omit<CatalogAlias, 'id'>> }>(`${this.apiDomain}/api/v1/catalog/items/${itemId}/aliases`, {
        data: {
          type: 'catalog_item_aliases',
          id: input.id,
          attributes: {
            scheme: input.scheme,
            value: input.value,
            party_id: input.party_id,
          },
        },
      })
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  removeAlias(itemId: string, aliasId: string): Observable<void> {
    return this.http.delete<void>(`${this.apiDomain}/api/v1/catalog/items/${itemId}/aliases/${aliasId}`);
  }

  listDuplicateClusters(): Observable<DuplicateCluster[]> {
    return this.http
      .get<{
        data: Array<{
          id: string;
          attributes: {
            reason: string;
            item_ids: string[];
            items: Array<{ id: string; attributes: Omit<DuplicateClusterMember, 'id'> }>;
          };
        }>;
      }>(`${this.apiDomain}/api/v1/catalog/items/duplicate-clusters`)
      .pipe(
        map((res) =>
          res.data.map((doc) => ({
            id: doc.id,
            reason: doc.attributes.reason,
            item_ids: doc.attributes.item_ids,
            items: (doc.attributes.items || []).map((item) => ({ id: item.id, ...item.attributes })),
          })),
        ),
      );
  }

  mergeItems(input: MergeItemsInput): Observable<CatalogItem> {
    return this.http
      .post<{ data: JsonApiDoc<Omit<CatalogItem, 'id'>> }>(`${this.apiDomain}/api/v1/catalog/items/merges`, {
        data: {
          type: 'catalog_item_merges',
          attributes: {
            survivor_id: input.survivor_id,
            source_ids: input.source_ids,
            name: input.name,
            kind: input.kind,
            internal_code: input.internal_code,
          },
        },
      })
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  markNotDuplicates(itemIds: string[]): Observable<void> {
    return this.http
      .post(`${this.apiDomain}/api/v1/catalog/items/not-duplicates`, { data: { type: 'catalog_item_not_duplicates', attributes: { item_ids: itemIds } } }, { responseType: 'text' })
      .pipe(map(() => undefined));
  }

  downloadImportTemplate(): Observable<Blob> {
    return this.http.get(`${this.apiDomain}/api/v1/catalog/imports/template`, { responseType: 'blob' });
  }

  queueImport(id: string, fileKey: string): Observable<CatalogImport> {
    return this.http
      .post<{ data: JsonApiDoc<Omit<CatalogImport, 'id'>> }>(`${this.apiDomain}/api/v1/catalog/imports`, {
        data: { type: 'catalog_imports', id, attributes: { file_key: fileKey } },
      })
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  listImports(pageSize = 20, after?: string): Observable<CatalogPage<CatalogImport>> {
    let params = new HttpParams().set('page[size]', String(pageSize));
    if (after) params = params.set('page[after]', after);
    return this.http.get<JsonApiList<Omit<CatalogImport, 'id'>>>(`${this.apiDomain}/api/v1/catalog/imports`, { params }).pipe(
      map((res) => ({
        items: res.data.map((doc) => ({ id: doc.id, ...doc.attributes })),
        hasMore: Boolean(res.meta?.has_more),
        cursor: res.meta?.cursor,
      })),
    );
  }

  getImport(id: string): Observable<CatalogImport> {
    return this.http.get<{ data: JsonApiDoc<Omit<CatalogImport, 'id'>> }>(`${this.apiDomain}/api/v1/catalog/imports/${id}`).pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  cancelImport(id: string): Observable<CatalogImport> {
    return this.http
      .post<{ data: JsonApiDoc<Omit<CatalogImport, 'id'>> }>(`${this.apiDomain}/api/v1/catalog/imports/${id}/cancel`, {})
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  listImportErrors(importId: string, pageSize = 50, after?: string): Observable<CatalogPage<CatalogImportError>> {
    let params = new HttpParams().set('page[size]', String(pageSize));
    if (after) params = params.set('page[after]', after);
    return this.http.get<JsonApiList<Omit<CatalogImportError, 'id'>>>(`${this.apiDomain}/api/v1/catalog/imports/${importId}/errors`, { params }).pipe(
      map((res) => ({
        items: res.data.map((doc) => ({ id: doc.id, ...doc.attributes })),
        hasMore: Boolean(res.meta?.has_more),
        cursor: res.meta?.cursor,
        total: res.meta?.total,
      })),
    );
  }
}
