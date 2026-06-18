import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { InvoiceListResponse, JsonApiCollectionResponse, JsonApiDocument, InvoiceSummary, InvoiceDetails } from '../domain/invoice.model';

@Injectable({ providedIn: 'root' })
export class InvoicesHttpService {
  private readonly http = inject(HttpClient);
  private readonly apiDomain = environment.apiUrl;

  listInvoices(limit: number = 20, cursor?: string): Observable<InvoiceListResponse> {
    let params = new HttpParams().set('limit', limit);
    if (cursor) {
      params = params.set('cursor', cursor);
    }
    return this.http.get<JsonApiCollectionResponse<Omit<InvoiceSummary, 'id'>>>(`${this.apiDomain}/api/v1/invoicing/invoices`, { params }).pipe(
      map((response) => ({
        items: response.data.map((doc) => ({ id: doc.id, ...doc.attributes })),
        has_more: response.meta.has_more,
        limit: response.meta.limit,
        cursor: response.meta.cursor,
      })),
    );
  }

  getInvoiceById(id: string): Observable<InvoiceDetails> {
    return this.http
      .get<{ data: JsonApiDocument<Omit<InvoiceDetails, 'id'>> }>(`${this.apiDomain}/api/v1/invoicing/invoices/${id}`)
      .pipe(map((response) => ({ id: response.data.id, ...response.data.attributes })));
  }
}
