import { Injectable, computed, inject, signal } from '@angular/core';
import { Subscription, catchError, finalize, forkJoin, interval, of, startWith, switchMap } from 'rxjs';
import { UNIFIED_INBOX_REPOSITORY } from '../domain/unified-inbox.repository';
import { MAIL_PROVIDERS, MailProvider, providerLabel } from '../domain/inbox.types';
import {
  AccountHealthSummary,
  MessageProcessingStatus,
  UnifiedInboxFilters,
  UnifiedInboxMessage,
} from '../domain/unified-inbox.model';

@Injectable({ providedIn: 'root' })
export class UnifiedInboxStore {
  readonly loading = signal(false);
  readonly messages = signal<UnifiedInboxMessage[]>([]);
  readonly accountHealth = signal<AccountHealthSummary[]>([]);

  readonly providers = MAIL_PROVIDERS;
  readonly statuses: MessageProcessingStatus[] = ['new', 'processed', 'skipped', 'error'];

  readonly filters = signal<UnifiedInboxFilters>({
    provider: 'all',
    status: 'all',
    onlyInvoices: false,
    search: '',
  });

  readonly filteredMessages = computed(() => {
    const activeFilters = this.filters();
    const normalizedSearch = activeFilters.search.trim().toLowerCase();

    return this.messages().filter((message) => {
      if (activeFilters.provider !== 'all' && message.provider !== activeFilters.provider) {
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

      return [message.subject, message.sender, message.account_email].some((value) =>
        (value || '').toLowerCase().includes(normalizedSearch),
      );
    });
  });

  private readonly repository = inject(UNIFIED_INBOX_REPOSITORY);
  private accountHealthSub?: Subscription;

  init(): void {
    this.loadData();
    this.startAccountHealthPolling();
  }

  destroy(): void {
    this.accountHealthSub?.unsubscribe();
  }

  patchFilters(partial: Partial<UnifiedInboxFilters>): void {
    this.filters.update((current) => ({ ...current, ...partial }));
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

    forkJoin({
      messages: this.repository.listMessages().pipe(catchError(() => of([] as UnifiedInboxMessage[]))),
      accounts: this.repository.listAccountHealth().pipe(catchError(() => of([] as AccountHealthSummary[]))),
    })
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe(({ messages, accounts }) => {
        this.messages.set(messages);
        this.accountHealth.set(accounts);
      });
  }

  private startAccountHealthPolling(): void {
    this.accountHealthSub?.unsubscribe();
    this.accountHealthSub = interval(30000)
      .pipe(
        startWith(0),
        switchMap(() => this.repository.listAccountHealth().pipe(catchError(() => of([] as AccountHealthSummary[])))),
      )
      .subscribe((accounts) => this.accountHealth.set(accounts));
  }
}
