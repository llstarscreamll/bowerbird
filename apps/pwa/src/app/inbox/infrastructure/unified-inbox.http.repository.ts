import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { environment } from '../../../environments/environment';
import { AccountHealthSummary, AccountSyncStatus, MessageListPage, SendMessagePayload, UnifiedInboxMessageDetail } from '../domain/unified-inbox.model';
import { MailProvider } from '../domain/inbox.types';
import { ListMessagesParams, UnifiedInboxRepository } from '../domain/unified-inbox.repository';

@Injectable({ providedIn: 'root' })
export class UnifiedInboxHttpRepository implements UnifiedInboxRepository {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/api/v1/inbox`;
  private readonly connectionsUrl = `${environment.apiUrl}/api/v1/connections`;

  listMessages(params?: ListMessagesParams): Observable<MessageListPage> {
    let httpParams = new HttpParams();
    if (params?.folder) {
      httpParams = httpParams.set('folder', params.folder);
    }
    if (params?.accountId && params.accountId !== 'all') {
      httpParams = httpParams.set('account_id', params.accountId);
    }
    if (params?.q) {
      httpParams = httpParams.set('q', params.q);
    }
    if (params?.limit) {
      httpParams = httpParams.set('limit', String(params.limit));
    }
    if (params?.offset) {
      httpParams = httpParams.set('offset', String(params.offset));
    }
    if (params?.onlyInvoices) {
      httpParams = httpParams.set('only_invoices', 'true');
    }

    return this.http.get<MessageListPage>(`${this.baseUrl}/messages`, { params: httpParams });
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

  triggerSync(): Observable<void> {
    return this.http.post(`${this.baseUrl}/sync`, {}, { responseType: 'text' }).pipe(map(() => void 0));
  }

  getProviderAuthUrl(provider: MailProvider): Observable<string> {
    const backendProvider = this.toBackendConnectionProvider(provider);
    return this.http.get<{ data: { auth_url: string } }>(`${this.connectionsUrl}/${backendProvider}`).pipe(map((response) => response.data.auth_url));
  }

  modifyMessage(messageId: string, action: 'read' | 'unread' | 'star' | 'unstar' | 'archive' | 'trash'): Observable<void> {
    return this.http.post(`${this.baseUrl}/messages/${messageId}/${action}`, {}, { responseType: 'text' }).pipe(map(() => void 0));
  }

  sendMessage(payload: SendMessagePayload): Observable<{ id: string }> {
    return this.http.post<{ id: string }>(`${this.baseUrl}/messages`, payload);
  }

  downloadAttachment(messageId: string, attachmentId: string): Observable<Blob> {
    return this.http.get(`${this.baseUrl}/messages/${messageId}/attachments/${attachmentId}`, { responseType: 'blob' });
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

  private toBackendConnectionProvider(provider: MailProvider): string {
    switch (provider) {
      case 'gmail':
        return 'google';
      case 'microsoft':
      case 'outlook':
      case 'hotmail':
        return 'microsoft';
      case 'yahoo':
        return 'yahoo';
      default:
        return provider;
    }
  }
}

interface ConnectionListItem {
  id: string;
  provider: string;
  provider_account_email: string;
  status: 'active' | 'requires_reconnect' | 'paused' | 'error';
}
