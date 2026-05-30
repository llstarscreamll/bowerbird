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
  provider_message?: ProviderMailMessage;
}

export interface ProviderMailHeader {
  name: string;
  value: string;
}

export interface ProviderMailPartBody {
  attachment_id?: string;
  data?: string;
  size: number;
}

export interface ProviderMailPart {
  part_id?: string;
  mime_type?: string;
  filename?: string;
  headers?: ProviderMailHeader[];
  body: ProviderMailPartBody;
  parts?: ProviderMailPart[];
}

export interface ProviderMailAttachmentRef {
  AttachmentID: string;
  Filename: string;
  MimeType: string;
  Size: number;
}

export interface ProviderMailMessage {
  id: string;
  thread_id: string;
  label_ids?: string[];
  subject: string;
  sender: string;
  snippet: string;
  plain_text_body: string;
  html_body?: string;
  headers?: ProviderMailHeader[];
  payload?: ProviderMailPart;
  history_id?: string;
  size_estimate?: number;
  received_at?: string;
  internal_date?: string;
  attachments?: ProviderMailAttachmentRef[];
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
