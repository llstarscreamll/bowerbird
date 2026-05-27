import { Injectable, inject, signal } from '@angular/core';
import { finalize } from 'rxjs';
import { Connection, ConnectionProvider, CONNECTION_PROVIDERS, providerLabel } from '../domain/connections.model';
import { CONNECTIONS_REPOSITORY } from '../domain/connections.repository';
import { ToastService } from '../../core/services/toast.service';

@Injectable({ providedIn: 'root' })
export class ConnectionsStore {
  readonly providers = CONNECTION_PROVIDERS;
  readonly connections = signal<Connection[]>([]);
  readonly selectedConnection = signal<Connection | null>(null);
  readonly loading = signal(false);
  readonly submitting = signal(false);
  readonly disconnectingId = signal<string | null>(null);
  readonly errorMessage = signal('');

  private readonly repository = inject(CONNECTIONS_REPOSITORY);
  private readonly toast = inject(ToastService);

  loadConnections(): void {
    this.loading.set(true);
    this.errorMessage.set('');

    this.repository
      .listConnections()
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe({
        next: (connections) => this.connections.set(connections),
        error: () => this.errorMessage.set('No fue posible cargar las conexiones.'),
      });
  }

  loadConnection(connectionId: string): void {
    this.loading.set(true);
    this.errorMessage.set('');

    this.repository
      .listConnections()
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe({
        next: (connections) => {
          this.connections.set(connections);
          const selected = connections.find((connection) => connection.id === connectionId) || null;
          this.selectedConnection.set(selected);

          if (!selected) {
            this.errorMessage.set('No fue posible encontrar la conexión solicitada.');
          }
        },
        error: () => this.errorMessage.set('No fue posible cargar los detalles de la conexión.'),
      });
  }

  updateSharingPolicy(connectionId: string, sharing_policy: 'private' | 'tenant_all'): void {
    this.submitting.set(true);
    this.errorMessage.set('');

    this.repository
      .updateConnection(connectionId, { sharing_policy })
      .pipe(finalize(() => this.submitting.set(false)))
      .subscribe({
        next: (connection) => {
          this.selectedConnection.set(connection);
          this.connections.update((list) => list.map((c) => (c.id === connectionId ? connection : c)));
          this.toast.showSuccess('Opciones de visibilidad actualizadas');
        },
        error: () => this.errorMessage.set('No fue posible actualizar las opciones de visibilidad.'),
      });
  }

  connectProvider(provider: ConnectionProvider, onAuthRedirect?: (url: string) => void): void {
    this.submitting.set(true);
    this.errorMessage.set('');

    this.repository
      .connectProvider(provider)
      .pipe(finalize(() => this.submitting.set(false)))
      .subscribe({
        next: (response) => {
          if (response.auth_url) {
            onAuthRedirect?.(response.auth_url);
            return;
          }
          this.loadConnections();
        },
        error: (err) => {
          this.errorMessage.set(err?.error?.message || 'No fue posible iniciar la conexión.');
        },
      });
  }

  disconnectConnection(connectionId: string, onSuccess?: () => void): void {
    this.disconnectingId.set(connectionId);
    this.errorMessage.set('');

    this.repository
      .disconnectConnection(connectionId)
      .pipe(finalize(() => this.disconnectingId.set(null)))
      .subscribe({
        next: () => {
          this.connections.update((list) => list.filter((item) => item.id !== connectionId));
          this.toast.showSuccess('Conexión eliminada exitosamente');
          if (onSuccess) onSuccess();
        },
        error: () => this.errorMessage.set('No fue posible eliminar la conexión.'),
      });
  }

  providerLabel(provider: ConnectionProvider): string {
    return providerLabel(provider);
  }
}
