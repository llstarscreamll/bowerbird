import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { environment } from '../../../environments/environment';
import { AccountHealthSummary, AccountSyncStatus, UnifiedInboxMessage, UnifiedInboxMessageDetail } from '../domain/unified-inbox.model';
import { MailProvider } from '../domain/inbox.types';
import { UnifiedInboxRepository } from '../domain/unified-inbox.repository';

@Injectable({ providedIn: 'root' })
export class UnifiedInboxHttpRepository implements UnifiedInboxRepository {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/api/v1/inbox`;
  private readonly connectionsUrl = `${environment.apiUrl}/api/v1/connections`;

  listMessages(): Observable<UnifiedInboxMessage[]> {
    return this.http.get<UnifiedInboxMessage[]>(`${this.baseUrl}/messages`);
  }

  getMessage(messageId: string): Observable<UnifiedInboxMessageDetail> {
    return this.http.get<UnifiedInboxMessageDetail>(`${this.baseUrl}/messages/${messageId}`);
  }

  listAccountHealth(): Observable<AccountHealthSummary[]> {
    return this.http.get<{ data: ConnectionListItem[] }>(this.connectionsUrl).pipe(
      map((response) =>
        response.data.map((connection) => ({
          id: connection.id,
          provider: this.normalizeProvider(connection.provider),
          email_address: connection.provider_account_email,
          status: connection.status,
          connection_status: connection.status,
        })),
      ),
    );
  }

  listAccountSyncStatus(): Observable<AccountSyncStatus[]> {
    return this.http.get<AccountSyncStatus[]>(`${this.baseUrl}/sync-status`);
  }

  triggerSync(accountId?: string): Observable<void> {
    const body = accountId && accountId !== 'all' ? { account_id: accountId } : {};
    return this.http.post<void>(`${this.baseUrl}/sync`, body);
  }

  private normalizeProvider(provider: string): MailProvider {
    switch (provider) {
      case 'google':
        return 'gmail';
      case 'gmail':
      case 'microsoft':
      case 'outlook':
      case 'hotmail':
      case 'yahoo':
        return provider;
      default:
        return 'gmail';
    }
  }
}

interface ConnectionListItem {
  id: string;
  provider: string;
  provider_account_email: string;
  status: 'active' | 'requires_reconnect' | 'paused' | 'error';
}
