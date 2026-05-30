import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';
import { AccountHealthSummary, AccountSyncStatus, UnifiedInboxMessage, UnifiedInboxMessageDetail } from './unified-inbox.model';
import { MailProvider } from './inbox.types';

export interface UnifiedInboxRepository {
  listMessages(): Observable<UnifiedInboxMessage[]>;
  getMessage(messageId: string): Observable<UnifiedInboxMessageDetail>;
  listAccountHealth(): Observable<AccountHealthSummary[]>;
  listAccountSyncStatus(): Observable<AccountSyncStatus[]>;
  triggerSync(): Observable<void>;
  getProviderAuthUrl(provider: MailProvider): Observable<string>;
}

export const UNIFIED_INBOX_REPOSITORY = new InjectionToken<UnifiedInboxRepository>('UNIFIED_INBOX_REPOSITORY');
