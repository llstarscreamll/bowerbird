export type SharingPolicy = 'private' | 'tenant_all';
export type ConnectionStatus = 'active' | 'requires_reconnect' | 'paused' | 'error';
export type ConnectionProvider = 'gmail' | 'microsoft';

export interface Connection {
  id: string;
  provider: ConnectionProvider;
  provider_account_email: string;
  status: ConnectionStatus;
  sharing_policy: SharingPolicy;
}

export interface ConnectAccountResponse {
  auth_url?: string;
  message?: string;
}

export const CONNECTION_PROVIDERS: ConnectionProvider[] = ['gmail', 'microsoft'];

export function providerLabel(provider: ConnectionProvider): string {
  switch (provider) {
    case 'gmail':
      return 'Google Workspace / Gmail';
    case 'microsoft':
      return 'Microsoft 365 / Outlook';
    default:
      return provider;
  }
}
