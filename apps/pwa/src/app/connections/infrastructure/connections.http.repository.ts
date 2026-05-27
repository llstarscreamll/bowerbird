import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { environment } from '../../../environments/environment';
import { Connection, ConnectAccountResponse, ConnectionProvider } from '../domain/connections.model';
import { ConnectionsRepository } from '../domain/connections.repository';

@Injectable({ providedIn: 'root' })
export class ConnectionsHttpRepository implements ConnectionsRepository {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/api/v1/connections`;

  listConnections(): Observable<Connection[]> {
    return this.http.get<{ data: Connection[] }>(this.baseUrl).pipe(map((res) => res.data));
  }

  getConnection(connectionId: string): Observable<Connection> {
    return this.http.get<{ data: Connection }>(`${this.baseUrl}/${encodeURIComponent(connectionId)}`).pipe(map((res) => res.data));
  }

  updateConnection(connectionId: string, payload: { sharing_policy: 'private' | 'tenant_all' }): Observable<Connection> {
    return this.http.patch<{ data: Connection }>(`${this.baseUrl}/${encodeURIComponent(connectionId)}`, payload).pipe(map((res) => res.data));
  }

  connectProvider(provider: ConnectionProvider): Observable<ConnectAccountResponse> {
    // Only google is supported right now in backend, adapt as needed.
    const backendProvider = provider === 'gmail' ? 'google' : provider;
    return this.http.get<{ data: ConnectAccountResponse }>(`${this.baseUrl}/${encodeURIComponent(backendProvider)}`).pipe(map((res) => res.data));
  }

  disconnectConnection(connectionId: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${encodeURIComponent(connectionId)}`);
  }
}
