import { ConnectionStatus, MailProvider } from './inbox.types';

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

export interface UnifiedInboxFilters {
  provider: 'all' | MailProvider;
  status: 'all' | MessageProcessingStatus;
  onlyInvoices: boolean;
  search: string;
}

export interface AccountHealthSummary {
  id: string;
  provider: MailProvider;
  email_address: string;
  status: ConnectionStatus;
  last_synced_at?: string;
}
