import { Injectable, computed, inject, signal } from '@angular/core';
import { Subscription, catchError, finalize, forkJoin, interval, of, switchMap } from 'rxjs';
import { resolveConnectionStatus } from '../../core/domain/connection-status.model';
import { MailProvider, MAIL_PROVIDERS } from '../domain/inbox.types';
import { extractSyncActionError, SyncActionError } from './sync-error-parser';
import { UNIFIED_INBOX_REPOSITORY } from '../domain/unified-inbox.repository';
import {
  AccountHealthSummary,
  AccountSyncStatus,
  MailFolder,
  MessageProcessingStatus,
  SendMessagePayload,
  UnifiedInboxFilters,
  UnifiedInboxMessage,
  UnifiedInboxMessageDetail,
} from '../domain/unified-inbox.model';

@Injectable({ providedIn: 'root' })
export class UnifiedInboxStore {
  private static readonly PAGE_SIZE = 50;

  readonly loading = signal(false);
  readonly isLoadingMore = signal(false);
  readonly error = signal<string | null>(null);
  readonly messages = signal<UnifiedInboxMessage[]>([]);
  readonly totalMessages = signal(0);
  readonly composeOpen = signal(false);
  readonly sending = signal(false);
  readonly accountHealth = signal<AccountHealthSummary[]>([]);
  readonly detailError = signal<string | null>(null);
  readonly loadingMessageId = signal<string | null>(null);
  readonly syncActionError = signal<SyncActionError | null>(null);
  readonly syncRetrySecondsLeft = signal(0);

  readonly hasMore = computed(() => this.messages().length < this.totalMessages());

  readonly providers = MAIL_PROVIDERS;
  readonly statuses: MessageProcessingStatus[] = ['new', 'processed', 'skipped', 'error'];

  readonly filters = signal<UnifiedInboxFilters>({
    provider: 'all',
    accountId: 'all',
    status: 'all',
    folder: 'inbox',
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
    const latestByThread = new Map<string, UnifiedInboxMessage>();

    for (const message of this.messages()) {
      if (activeFilters.provider !== 'all' && message.provider !== activeFilters.provider) {
        continue;
      }
      if (activeFilters.status !== 'all' && message.processing_status !== activeFilters.status) {
        continue;
      }

      const threadKey = message.thread_id || message.id;
      const current = latestByThread.get(threadKey);
      if (!current || current.received_at < message.received_at) {
        latestByThread.set(threadKey, message);
      }
    }

    return Array.from(latestByThread.values());
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

  triggerSync(): void {
    if (this.syncRetrySecondsLeft() > 0) {
      return;
    }

    this.repository
      .triggerSync()
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
    if (partial.folder || partial.accountId || partial.search !== undefined || partial.onlyInvoices !== undefined) {
      this.reloadMessages();
    }
  }

  setFolder(folder: MailFolder): void {
    this.patchFilters({ folder });
  }

  modifyMessage(messageId: string, action: 'read' | 'unread' | 'star' | 'unstar' | 'archive' | 'trash'): void {
    this.repository.modifyMessage(messageId, action).subscribe({
      next: () => this.reloadMessages(),
      error: () => this.error.set('No se pudo actualizar el mensaje.'),
    });
  }

  sendMessage(payload: SendMessagePayload): void {
    this.sending.set(true);
    this.repository
      .sendMessage(payload)
      .pipe(finalize(() => this.sending.set(false)))
      .subscribe({
        next: () => {
          this.composeOpen.set(false);
          this.reloadMessages();
        },
        error: () => this.error.set('No se pudo enviar el correo.'),
      });
  }

  downloadAttachment(messageId: string, attachmentId: string, filename: string): void {
    this.repository.downloadAttachment(messageId, attachmentId).subscribe({
      next: (blob) => {
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = filename;
        link.click();
        URL.revokeObjectURL(url);
      },
      error: () => this.error.set('No se pudo descargar el adjunto.'),
    });
  }

  loadMore(): void {
    if (!this.hasMore() || this.isLoadingMore() || this.loading()) {
      return;
    }

    this.isLoadingMore.set(true);
    const offset = this.messages().length;

    this.repository
      .listMessages(this.messageListParams(offset))
      .pipe(
        catchError(() => of({ data: [] as UnifiedInboxMessage[], total: this.totalMessages(), limit: UnifiedInboxStore.PAGE_SIZE, offset })),
        finalize(() => this.isLoadingMore.set(false)),
      )
      .subscribe((page) => {
        const incoming = page.data ?? [];
        if (incoming.length === 0) {
          this.totalMessages.set(this.messages().length);
          return;
        }

        let appended = 0;
        this.messages.update((current) => {
          const seen = new Set(current.map((message) => message.id));
          const unique = incoming.filter((message) => !seen.has(message.id));
          appended = unique.length;
          return [...current, ...unique];
        });

        if (appended === 0) {
          this.totalMessages.set(this.messages().length);
          return;
        }

        this.totalMessages.set(page.total ?? this.totalMessages());
      });
  }

  private reloadMessages(): void {
    this.repository
      .listMessages(this.messageListParams())
      .pipe(catchError(() => of({ data: [] as UnifiedInboxMessage[], total: 0, limit: UnifiedInboxStore.PAGE_SIZE, offset: 0 })))
      .subscribe((page) => {
        this.messages.set(page.data ?? []);
        this.totalMessages.set(page.total ?? 0);
      });
  }

  private messageListParams(offset = 0, limit = UnifiedInboxStore.PAGE_SIZE) {
    const filters = this.filters();
    return {
      folder: filters.folder,
      accountId: filters.accountId,
      q: filters.search,
      onlyInvoices: filters.onlyInvoices,
      limit,
      offset,
    };
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

  private loadData(): void {
    this.loading.set(true);
    this.error.set(null);

    forkJoin({
      messages: this.repository.listMessages(this.messageListParams()).pipe(
        catchError(() => {
          this.error.set('No se pudieron cargar los mensajes. Por favor, inténtelo de nuevo más tarde.');
          return of({ data: [] as UnifiedInboxMessage[], total: 0, limit: UnifiedInboxStore.PAGE_SIZE, offset: 0 });
        }),
      ),
      accounts: this.repository.listAccountHealth().pipe(
        catchError(() => {
          this.error.set('No se pudieron cargar las cuentas conectadas. Por favor, inténtelo de nuevo más tarde.');
          return of([] as AccountHealthSummary[]);
        }),
      ),
      syncStatus: this.repository.listAccountSyncStatus().pipe(catchError(() => of([] as AccountSyncStatus[]))),
    })
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe(({ messages, accounts, syncStatus }) => {
        this.messages.set(messages.data ?? []);
        this.totalMessages.set(messages.total ?? 0);
        this.accountHealth.set(this.mergeAccountHealthWithSyncStatus(accounts, syncStatus));

        this.triggerSync();

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

    // Poll first page only and merge so pages loaded via infinite scroll are kept.
    this.messagePollSub = interval(30000)
      .pipe(
        switchMap(() =>
          this.repository.listMessages(this.messageListParams()).pipe(
            catchError(() => {
              return of({ data: [] as UnifiedInboxMessage[], total: this.totalMessages(), limit: UnifiedInboxStore.PAGE_SIZE, offset: 0 });
            }),
          ),
        ),
      )
      .subscribe((page) => {
        const fresh = page.data ?? [];
        if (fresh.length === 0 && page.total !== 0) {
          return;
        }

        this.totalMessages.set(page.total ?? 0);
        this.messages.update((current) => this.mergePolledMessages(current, fresh));
      });
  }

  private mergePolledMessages(current: UnifiedInboxMessage[], fresh: UnifiedInboxMessage[]): UnifiedInboxMessage[] {
    if (current.length === 0 || current.length <= UnifiedInboxStore.PAGE_SIZE) {
      return fresh;
    }

    const freshIds = new Set(fresh.map((message) => message.id));
    const olderPages = current.filter((message) => !freshIds.has(message.id));
    return [...fresh, ...olderPages];
  }

  private mergeAccountHealthWithSyncStatus(accounts: AccountHealthSummary[], syncStatus: AccountSyncStatus[]): AccountHealthSummary[] {
    const syncStatusByAccount = new Map(syncStatus.map((status) => [status.id, status]));

    return accounts.map((account) => {
      const currentSyncStatus = syncStatusByAccount.get(account.id);

      return {
        ...account,
        status: currentSyncStatus?.status ?? account.status,
        connection_status: account.connection_status ?? resolveConnectionStatus({ status: account.status }),
        sync_status: currentSyncStatus?.status,
        last_synced_at: currentSyncStatus?.last_synced_at,
      };
    });
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
