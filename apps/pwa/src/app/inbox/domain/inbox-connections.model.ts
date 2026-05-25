import { ConnectionStatus, MailProvider } from './inbox.types';

export interface ConnectedAccount {
  id: string;
  provider: MailProvider;
  email_address: string;
  status: ConnectionStatus;
  last_synced_at?: string;
  last_error?: string;
  created_at: string;
}

export interface ConnectAccountRequest {
  provider: MailProvider;
  email_address: string;
}

export interface ConnectAccountResponse {
  account: ConnectedAccount;
  auth_url?: string;
}
