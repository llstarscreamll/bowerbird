import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { InvoiceHistoryAnalyzeFileReference, StartInvoiceHistoryAnalysisRequest } from '../domain/invoice-history-analysis.model';

@Injectable({ providedIn: 'root' })
export class InvoiceHistoryHttpService {
  private readonly http = inject(HttpClient);
  private readonly apiDomain = environment.apiUrl;

  startAnalysis(requestId: string, files: readonly InvoiceHistoryAnalyzeFileReference[]): Observable<void> {
    const payload: StartInvoiceHistoryAnalysisRequest = {
      data: {
        id: requestId,
        type: 'queue-invoice-extraction',
        attributes: {
          files: files.map((file) => ({
            name: file.name,
            path: file.url,
            mime_type: file.type,
          })),
        },
      },
    };

    return this.http.post(`${this.apiDomain}/api/v1/invoicing/extractions`, payload, { responseType: 'text' }).pipe(map(() => void 0));
  }
}
