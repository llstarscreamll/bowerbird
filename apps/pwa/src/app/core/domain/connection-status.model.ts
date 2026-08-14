export type ConnectionStatus = 'active' | 'requires_reconnect' | 'paused' | 'error';

export type ConnectionStatusSource = {
  connection_status?: ConnectionStatus;
  status?: string;
};

export function resolveConnectionStatus(source: ConnectionStatusSource): ConnectionStatus | undefined {
  if (source.connection_status) {
    return source.connection_status;
  }

  const status = source.status;
  if (status === 'active' || status === 'requires_reconnect' || status === 'paused' || status === 'error') {
    return status;
  }

  return undefined;
}
