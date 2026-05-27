import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';
import { Connection, ConnectAccountResponse, ConnectionProvider } from './connections.model';

export interface ConnectionsRepository {
  listConnections(): Observable<Connection[]>;
  getConnection(connectionId: string): Observable<Connection>;
  connectProvider(provider: ConnectionProvider): Observable<ConnectAccountResponse>;
  updateConnection(connectionId: string, payload: { sharing_policy: 'private' | 'tenant_all' }): Observable<Connection>;
  disconnectConnection(connectionId: string): Observable<void>;
}

export const CONNECTIONS_REPOSITORY = new InjectionToken<ConnectionsRepository>('ConnectionsRepository');
