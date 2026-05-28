import { ConnectionStatus, MailProvider, SyncStatus } from './inbox.types';

export type MessageProcessingStatus = 'new' | 'processed' | 'skipped' | 'error';

export interface UnifiedInboxMessage {
  id: string;
  provider: MailProvider;
  account_id: string;
  account_email: string;
  subject: string;
  sender: string;
  snippet?: string;
  received_at: string;
  processing_status: MessageProcessingStatus;
  has_xml: boolean;
  has_pdf: boolean;
}

export interface UnifiedInboxMessageDetail extends UnifiedInboxMessage {
  body_text?: string;
  body_html?: string;
}

export interface UnifiedInboxFilters {
  provider: 'all' | MailProvider;
  accountId: string | 'all';
  status: 'all' | MessageProcessingStatus;
  onlyInvoices: boolean;
  search: string;
}

export interface AccountHealthSummary {
  id: string;
  provider: MailProvider;
  email_address: string;
  status: SyncStatus | ConnectionStatus;
  connection_status?: ConnectionStatus;
  sync_status?: SyncStatus;
  last_synced_at?: string;
}

export interface AccountSyncStatus {
  id: string;
  status: SyncStatus;
  last_synced_at?: string;
}
