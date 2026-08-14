import { resolveConnectionStatus } from './connection-status.model';
import { connectionStatusBadgeLabel, connectionStatusBadgeVariant, connectionStatusDetailLabel, connectionStatusIcon } from '../presentation/connection-status';

describe('connection status modularity', () => {
  it('resolves connection status from API fields in domain layer', () => {
    expect(resolveConnectionStatus({ connection_status: 'active' })).toBe('active');
    expect(resolveConnectionStatus({ status: 'syncing' })).toBeUndefined();
    expect(resolveConnectionStatus({ status: 'error' })).toBe('error');
  });

  it('maps presentation labels from resolved status', () => {
    expect(connectionStatusBadgeLabel('active')).toBe('Activa');
    expect(connectionStatusDetailLabel('active')).toBe('Conexión activa');
    expect(connectionStatusBadgeVariant('error')).toBe('destructive');
    expect(connectionStatusIcon('paused')).toBe('lucideCirclePause');
  });
});
