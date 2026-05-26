import { Injectable, inject, signal } from '@angular/core';
import { finalize } from 'rxjs';
import { ConnectAccountRequest, ConnectedAccount } from '../domain/inbox-connections.model';
import { INBOX_CONNECTIONS_REPOSITORY } from '../domain/inbox-connections.repository';
import { MAIL_PROVIDERS, MailProvider, providerLabel } from '../domain/inbox.types';
import { ToastService } from '../../core/services/toast.service';

@Injectable({ providedIn: 'root' })
export class InboxConnectionsStore {
  readonly providers = MAIL_PROVIDERS;
  readonly accounts = signal<ConnectedAccount[]>([]);
  readonly loading = signal(false);
  readonly submitting = signal(false);
  readonly disconnectingId = signal<string | null>(null);
  readonly errorMessage = signal('');

  private readonly repository = inject(INBOX_CONNECTIONS_REPOSITORY);
  private readonly toast = inject(ToastService);

  loadAccounts(): void {
    this.loading.set(true);
    this.errorMessage.set('');

    this.repository
      .listAccounts()
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe({
        next: (accounts) => this.accounts.set(accounts),
        error: () => this.errorMessage.set('No fue posible cargar las cuentas conectadas.'),
      });
  }

  connectAccount(provider: MailProvider, emailAddress: string, onAuthRedirect?: (url: string) => void): void {
    const normalizedEmail = emailAddress.trim().toLowerCase();
    if (!normalizedEmail) {
      return;
    }

    this.submitting.set(true);
    this.errorMessage.set('');

    const request: ConnectAccountRequest = {
      provider,
      email_address: normalizedEmail,
    };

    this.repository
      .connectAccount(request)
      .pipe(finalize(() => this.submitting.set(false)))
      .subscribe({
        next: (response) => {
          if (response.auth_url) {
            onAuthRedirect?.(response.auth_url);
            return;
          }
          this.loadAccounts();
        },
        error: (err) => {
          this.errorMessage.set(err?.error?.message || 'No fue posible conectar la cuenta.');
        },
      });
  }

  disconnectAccount(accountID: string): void {
    this.disconnectingId.set(accountID);
    this.errorMessage.set('');

    this.repository
      .disconnectAccount(accountID)
      .pipe(finalize(() => this.disconnectingId.set(null)))
      .subscribe({
        next: () => {
          this.accounts.update((list) => list.filter((item) => item.id !== accountID));
          this.toast.showSuccess('Cuenta desconectada exitosamente');
        },
        error: () => this.errorMessage.set('No fue posible desconectar la cuenta.'),
      });
  }

  providerLabel(provider: MailProvider): string {
    return providerLabel(provider);
  }
}
