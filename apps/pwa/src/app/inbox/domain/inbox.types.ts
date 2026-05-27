export type MailProvider = 'gmail' | 'microsoft' | 'outlook' | 'yahoo' | 'hotmail';

export type ConnectionStatus = 'active' | 'requires_reconnect' | 'paused' | 'error';
export type SyncStatus = 'idle' | 'syncing' | 'error';

export const MAIL_PROVIDERS: MailProvider[] = ['gmail', 'microsoft', 'outlook', 'yahoo', 'hotmail'];

export function providerLabel(provider: MailProvider): string {
  switch (provider) {
    case 'gmail':
      return 'Gmail';
    case 'microsoft':
      return 'Microsoft';
    case 'outlook':
      return 'Outlook';
    case 'hotmail':
      return 'Hotmail';
    case 'yahoo':
      return 'Yahoo';
    default:
      return provider;
  }
}
