import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ConnectAccountRequest, ConnectAccountResponse, ConnectedAccount } from '../domain/inbox-connections.model';
import { InboxConnectionsRepository } from '../domain/inbox-connections.repository';

@Injectable({ providedIn: 'root' })
export class InboxConnectionsHttpRepository implements InboxConnectionsRepository {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/api/v1/inbox/accounts`;

  listAccounts(): Observable<ConnectedAccount[]> {
    return this.http.get<ConnectedAccount[]>(this.baseUrl);
  }

  connectAccount(request: ConnectAccountRequest): Observable<ConnectAccountResponse> {
    return this.http.post<ConnectAccountResponse>(this.baseUrl, request);
  }

  disconnectAccount(accountId: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${encodeURIComponent(accountId)}`);
  }
}
