import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { AccountHealthSummary, UnifiedInboxMessage } from '../domain/unified-inbox.model';
import { UnifiedInboxRepository } from '../domain/unified-inbox.repository';

@Injectable({ providedIn: 'root' })
export class UnifiedInboxHttpRepository implements UnifiedInboxRepository {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/api/v1/inbox`;

  listMessages(): Observable<UnifiedInboxMessage[]> {
    return this.http.get<UnifiedInboxMessage[]>(`${this.baseUrl}/messages`);
  }

  listAccountHealth(): Observable<AccountHealthSummary[]> {
    return this.http.get<AccountHealthSummary[]>(`${this.baseUrl}/accounts/health`);
  }
}
