import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { InvoiceHistoryAnalyzeFileReference, StartInvoiceHistoryAnalysisRequest } from '../domain/invoice-history-analysis.model';

@Injectable({ providedIn: 'root' })
export class InvoiceHistoryHttpService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/api/v1/invoices/history`;

  startAnalysis(files: readonly InvoiceHistoryAnalyzeFileReference[]): Observable<void> {
    const payload: StartInvoiceHistoryAnalysisRequest = {
      files: [...files],
    };

    return this.http.post(`${this.baseUrl}/analyze`, payload, { responseType: 'text' }).pipe(map(() => void 0));
  }
}
