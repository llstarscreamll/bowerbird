import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';
import { ConnectAccountRequest, ConnectAccountResponse, ConnectedAccount } from './inbox-connections.model';

export interface InboxConnectionsRepository {
  listAccounts(): Observable<ConnectedAccount[]>;
  connectAccount(request: ConnectAccountRequest): Observable<ConnectAccountResponse>;
  disconnectAccount(accountId: string): Observable<void>;
}

export const INBOX_CONNECTIONS_REPOSITORY = new InjectionToken<InboxConnectionsRepository>(
  'INBOX_CONNECTIONS_REPOSITORY',
);
