import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';
import { AccountHealthSummary, AccountSyncStatus, MailFolder, MessageListPage, SendMessagePayload, UnifiedInboxMessageDetail } from './unified-inbox.model';
import { MailProvider } from './inbox.types';

export interface ListMessagesParams {
  folder?: MailFolder;
  accountId?: string;
  q?: string;
  limit?: number;
  offset?: number;
  onlyInvoices?: boolean;
}

export interface UnifiedInboxRepository {
  listMessages(params?: ListMessagesParams): Observable<MessageListPage>;
  getMessage(messageId: string): Observable<UnifiedInboxMessageDetail>;
  listAccountHealth(): Observable<AccountHealthSummary[]>;
  listAccountSyncStatus(): Observable<AccountSyncStatus[]>;
  triggerSync(): Observable<void>;
  getProviderAuthUrl(provider: MailProvider): Observable<string>;
  modifyMessage(messageId: string, action: 'read' | 'unread' | 'star' | 'unstar' | 'archive' | 'trash'): Observable<void>;
  sendMessage(payload: SendMessagePayload): Observable<{ id: string }>;
  downloadAttachment(messageId: string, attachmentId: string): Observable<Blob>;
}

export const UNIFIED_INBOX_REPOSITORY = new InjectionToken<UnifiedInboxRepository>('UNIFIED_INBOX_REPOSITORY');
