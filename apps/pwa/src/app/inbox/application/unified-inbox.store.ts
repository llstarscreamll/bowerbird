import { Injectable, computed, inject, signal } from '@angular/core';
import { Subscription, catchError, finalize, forkJoin, interval, of, switchMap } from 'rxjs';
import { MailProvider, MAIL_PROVIDERS, providerLabel } from '../domain/inbox.types';
import { extractSyncActionError, SyncActionError } from './sync-error-parser';
import { UNIFIED_INBOX_REPOSITORY } from '../domain/unified-inbox.repository';
import { AccountHealthSummary, AccountSyncStatus, MessageProcessingStatus, UnifiedInboxFilters, UnifiedInboxMessage, UnifiedInboxMessageDetail } from '../domain/unified-inbox.model';

@Injectable({ providedIn: 'root' })
export class UnifiedInboxStore {
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);
  readonly messages = signal<UnifiedInboxMessage[]>([]);
  readonly accountHealth = signal<AccountHealthSummary[]>([]);
  readonly detailError = signal<string | null>(null);
  readonly loadingMessageId = signal<string | null>(null);
  readonly syncActionError = signal<SyncActionError | null>(null);
  readonly syncRetrySecondsLeft = signal(0);

  readonly providers = MAIL_PROVIDERS;
  readonly statuses: MessageProcessingStatus[] = ['new', 'processed', 'skipped', 'error'];

  readonly filters = signal<UnifiedInboxFilters>({
    provider: 'all',
    accountId: 'all',
    status: 'all',
    onlyInvoices: false,
    search: '',
  });

  readonly isSyncing = computed(() => {
    const targetAccountId = this.filters().accountId;
    if (targetAccountId === 'all') {
      return this.accountHealth().some((acc) => acc.status === 'syncing');
    }
    const acc = this.accountHealth().find((a) => a.id === targetAccountId);
    return acc ? acc.status === 'syncing' : false;
  });

  readonly filteredMessages = computed(() => {
    const activeFilters = this.filters();
    const normalizedSearch = activeFilters.search.trim().toLowerCase();

    return this.messages().filter((message) => {
      if (activeFilters.provider !== 'all' && message.provider !== activeFilters.provider) {
        return false;
      }

      if (activeFilters.accountId !== 'all' && message.account_id !== activeFilters.accountId) {
        return false;
      }

      if (activeFilters.status !== 'all' && message.processing_status !== activeFilters.status) {
        return false;
      }

      if (activeFilters.onlyInvoices && !message.has_xml && !message.has_pdf) {
        return false;
      }

      if (!normalizedSearch) {
        return true;
      }

      return [message.subject, message.sender, message.account_email].some((value) => (value || '').toLowerCase().includes(normalizedSearch));
    });
  });

  private readonly repository = inject(UNIFIED_INBOX_REPOSITORY);
  private accountHealthSub?: Subscription;
  private messagePollSub?: Subscription;
  private retryCountdownSub?: Subscription;
  private readonly messageDetails = signal<Record<string, UnifiedInboxMessageDetail>>({});

  init(): void {
    this.loadData();
    this.startPolling();
  }

  destroy(): void {
    this.accountHealthSub?.unsubscribe();
    this.messagePollSub?.unsubscribe();
    this.retryCountdownSub?.unsubscribe();
  }

  triggerSync(accountId?: string): void {
    if (this.syncRetrySecondsLeft() > 0) {
      return;
    }

    const targetAccountId = accountId || this.filters().accountId;

    const requestedAccounts = targetAccountId === 'all' ? this.accountHealth() : this.accountHealth().filter((account) => account.id === targetAccountId);

    const activeAccounts = requestedAccounts.filter((account) => this.isActiveAccount(account));
    const syncableAccounts = activeAccounts.filter((account) => account.status !== 'syncing');

    if (syncableAccounts.length === 0) {
      return;
    }

    const syncableAccountIds = new Set(syncableAccounts.map((account) => account.id));

    // Optimistically update status to 'syncing' for targeted accounts
    this.accountHealth.update((accounts) =>
      accounts.map((acc) => {
        if (syncableAccountIds.has(acc.id)) {
          return { ...acc, status: 'syncing', sync_status: 'syncing' };
        }
        return acc;
      }),
    );

    forkJoin(syncableAccounts.map((account) => this.repository.triggerSync(account.id)))
      .pipe(
        catchError((error: unknown) => {
          const syncError = extractSyncActionError(error);
          if (syncError) {
            this.syncActionError.set(syncError);
            this.error.set(null);
            this.startRetryCountdown(syncError.retryAfterSeconds);
            this.logSyncEvent('sync_error_classified', {
              error_code: syncError.code,
              trace_id: syncError.traceId,
              provider: syncError.provider,
              requires_reauth: syncError.requiresReauth,
              retry_after_seconds: syncError.retryAfterSeconds,
            });

            if (syncError.requiresReauth) {
              this.logSyncEvent('sync_reauth_required', {
                error_code: syncError.code,
                trace_id: syncError.traceId,
                provider: syncError.provider,
              });
            }

            if (syncError.retryAfterSeconds > 0) {
              this.logSyncEvent('sync_rate_limited', {
                error_code: syncError.code,
                trace_id: syncError.traceId,
                provider: syncError.provider,
                retry_after_seconds: syncError.retryAfterSeconds,
              });
            }
          } else {
            this.error.set('No se pudo iniciar la sincronización.');
          }

          // Revert status on error by forcing a health check
          this.refreshHealth();
          return of(null);
        }),
      )
      .subscribe();
  }

  refreshHealth(): void {
    this.repository
      .listAccountSyncStatus()
      .pipe(catchError(() => of([] as AccountSyncStatus[])))
      .subscribe((syncStatus) => {
        this.accountHealth.update((accounts) => this.mergeAccountHealthWithSyncStatus(accounts, syncStatus));
      });
  }

  patchFilters(partial: Partial<UnifiedInboxFilters>): void {
    this.filters.update((current) => ({ ...current, ...partial }));
  }

  clearError(): void {
    this.error.set(null);
  }

  clearSyncActionError(): void {
    this.syncActionError.set(null);
    this.stopRetryCountdown();
  }

  reauthenticateProvider(provider: string | undefined, onFallback: () => void): void {
    const normalizedProvider = this.normalizeProvider(provider);
    if (!normalizedProvider) {
      onFallback();
      return;
    }

    this.repository.getProviderAuthUrl(normalizedProvider).subscribe({
      next: (authUrl) => {
        this.logSyncEvent('sync_reauth_redirect_started', {
          provider: normalizedProvider.toUpperCase(),
          trace_id: this.syncActionError()?.traceId,
          error_code: this.syncActionError()?.code,
        });
        window.location.assign(authUrl);
      },
      error: () => {
        this.error.set('No se pudo iniciar la reconexión automática. Continúa desde Conexiones.');
        onFallback();
      },
    });
  }

  loadMessageDetail(messageId: string): void {
    if (!messageId || this.messageDetails()[messageId]) {
      return;
    }

    this.detailError.set(null);
    this.loadingMessageId.set(messageId);

    this.repository
      .getMessage(messageId)
      .pipe(
        catchError(() => {
          this.detailError.set('No se pudieron cargar los detalles del correo.');
          return of(null);
        }),
        finalize(() => {
          if (this.loadingMessageId() === messageId) {
            this.loadingMessageId.set(null);
          }
        }),
      )
      .subscribe((detail) => {
        if (!detail) {
          return;
        }

        this.messageDetails.update((current) => ({ ...current, [messageId]: detail }));
      });
  }

  getMessageDetail(messageId: string): UnifiedInboxMessageDetail | null {
    if (!messageId) {
      return null;
    }

    return this.messageDetails()[messageId] ?? null;
  }

  providerLabel(provider: MailProvider): string {
    return providerLabel(provider);
  }

  messageStatusLabel(status: MessageProcessingStatus): string {
    switch (status) {
      case 'new':
        return 'Nuevo';
      case 'processed':
        return 'Procesado';
      case 'skipped':
        return 'Omitido';
      case 'error':
        return 'Error';
      default:
        return status;
    }
  }

  messageStatusClasses(status: MessageProcessingStatus): string {
    switch (status) {
      case 'new':
        return 'border-sky-200 bg-sky-50 text-sky-700';
      case 'processed':
        return 'border-emerald-200 bg-emerald-50 text-emerald-700';
      case 'skipped':
        return 'border-slate-200 bg-slate-50 text-slate-700';
      case 'error':
        return 'border-rose-200 bg-rose-50 text-rose-700';
      default:
        return 'border-slate-200 bg-slate-50 text-slate-700';
    }
  }

  providerClasses(provider: MailProvider): string {
    switch (provider) {
      case 'gmail':
        return 'border-red-200 bg-red-50 text-red-700';
      case 'microsoft':
      case 'outlook':
      case 'hotmail':
        return 'border-blue-200 bg-blue-50 text-blue-700';
      case 'yahoo':
        return 'border-violet-200 bg-violet-50 text-violet-700';
      default:
        return 'border-slate-200 bg-slate-50 text-slate-700';
    }
  }

  private loadData(): void {
    this.loading.set(true);
    this.error.set(null);

    forkJoin({
      messages: this.repository.listMessages().pipe(
        catchError((err) => {
          this.error.set('No se pudieron cargar los mensajes. Por favor, inténtelo de nuevo más tarde.');
          return of([] as UnifiedInboxMessage[]);
        }),
      ),
      accounts: this.repository.listAccountHealth().pipe(
        catchError((err) => {
          this.error.set('No se pudieron cargar las cuentas conectadas. Por favor, inténtelo de nuevo más tarde.');
          return of([] as AccountHealthSummary[]);
        }),
      ),
      syncStatus: this.repository.listAccountSyncStatus().pipe(catchError(() => of([] as AccountSyncStatus[]))),
    })
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe(({ messages, accounts, syncStatus }) => {
        this.messages.set(messages);
        this.accountHealth.set(this.mergeAccountHealthWithSyncStatus(accounts, syncStatus));

        this.triggerSync('all');

        // Auto-select the single account if there's exactly one
        if (accounts.length === 1 && this.filters().accountId === 'all') {
          this.patchFilters({ accountId: accounts[0].id });
        }
      });
  }

  private startPolling(): void {
    this.accountHealthSub?.unsubscribe();
    this.messagePollSub?.unsubscribe();

    // Poll health status more frequently if syncing
    this.accountHealthSub = interval(10000)
      .pipe(
        switchMap(() =>
          forkJoin({
            accounts: this.repository.listAccountHealth().pipe(catchError(() => of([] as AccountHealthSummary[]))),
            syncStatus: this.repository.listAccountSyncStatus().pipe(catchError(() => of([] as AccountSyncStatus[]))),
          }),
        ),
      )
      .subscribe(({ accounts, syncStatus }) => {
        this.accountHealth.set(this.mergeAccountHealthWithSyncStatus(accounts, syncStatus));
      });

    // Poll messages
    this.messagePollSub = interval(30000)
      .pipe(
        switchMap(() =>
          this.repository.listMessages().pipe(
            catchError((err) => {
              return of([] as UnifiedInboxMessage[]);
            }),
          ),
        ),
      )
      .subscribe((messages) => {
        if (messages.length > 0) {
          this.messages.set(messages);
        }
      });
  }

  private mergeAccountHealthWithSyncStatus(accounts: AccountHealthSummary[], syncStatus: AccountSyncStatus[]): AccountHealthSummary[] {
    const syncStatusByAccount = new Map(syncStatus.map((status) => [status.id, status]));

    return accounts.map((account) => {
      const currentSyncStatus = syncStatusByAccount.get(account.id);

      return {
        ...account,
        status: currentSyncStatus?.status ?? account.status,
        connection_status: account.connection_status ?? this.toConnectionStatus(account.status),
        sync_status: currentSyncStatus?.status,
        last_synced_at: currentSyncStatus?.last_synced_at,
      };
    });
  }

  private isActiveAccount(account: AccountHealthSummary): boolean {
    return (account.connection_status ?? this.toConnectionStatus(account.status)) === 'active';
  }

  private toConnectionStatus(status: AccountHealthSummary['status']): 'active' | 'requires_reconnect' | 'paused' | 'error' | undefined {
    if (status === 'active' || status === 'requires_reconnect' || status === 'paused' || status === 'error') {
      return status;
    }

    return undefined;
  }

  private startRetryCountdown(seconds: number): void {
    this.stopRetryCountdown();
    if (seconds <= 0) {
      this.syncRetrySecondsLeft.set(0);
      return;
    }

    this.syncRetrySecondsLeft.set(seconds);
    this.retryCountdownSub = interval(1000).subscribe(() => {
      const nextSeconds = this.syncRetrySecondsLeft() - 1;
      if (nextSeconds <= 0) {
        this.syncRetrySecondsLeft.set(0);
        this.retryCountdownSub?.unsubscribe();
        return;
      }

      this.syncRetrySecondsLeft.set(nextSeconds);
    });
  }

  private stopRetryCountdown(): void {
    this.retryCountdownSub?.unsubscribe();
    this.retryCountdownSub = undefined;
    this.syncRetrySecondsLeft.set(0);
  }

  private normalizeProvider(provider: string | undefined): MailProvider | null {
    const value = (provider || '').trim().toLowerCase();
    switch (value) {
      case 'gmail':
      case 'microsoft':
      case 'outlook':
      case 'hotmail':
      case 'yahoo':
        return value;
      default:
        return null;
    }
  }

  private logSyncEvent(eventName: string, payload: Record<string, unknown>): void {
    console.info(`[${eventName}]`, payload);
  }
}
