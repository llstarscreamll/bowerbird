import { ConnectionStatus } from '../../domain/connection-status.model';

export function connectionStatusBadgeLabel(status: ConnectionStatus | undefined): string {
  switch (status) {
    case 'active':
      return 'Activa';
    case 'requires_reconnect':
      return 'Reconexión';
    case 'paused':
      return 'Pausada';
    case 'error':
      return 'Error';
    default:
      return status ?? '';
  }
}

export function connectionStatusDetailLabel(status: ConnectionStatus | undefined): string {
  switch (status) {
    case 'active':
      return 'Conexión activa';
    case 'requires_reconnect':
      return 'Requiere reconexión';
    case 'paused':
      return 'Conexión pausada';
    case 'error':
      return 'Conexión con error';
    default:
      return 'Estado de conexión desconocido';
  }
}

export function connectionStatusBadgeVariant(status: ConnectionStatus | undefined): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'active':
      return 'default';
    case 'requires_reconnect':
      return 'secondary';
    case 'error':
      return 'destructive';
    default:
      return 'outline';
  }
}

export function connectionStatusIcon(status: ConnectionStatus | undefined): string {
  switch (status) {
    case 'active':
      return 'lucideCircleCheck';
    case 'requires_reconnect':
      return 'lucideRefreshCwOff';
    case 'paused':
      return 'lucideCirclePause';
    case 'error':
      return 'lucideCircleAlert';
    default:
      return 'lucideCircleHelp';
  }
}

export function connectionStatusIconClasses(status: ConnectionStatus | undefined): string {
  switch (status) {
    case 'active':
      return 'text-emerald-500 dark:text-emerald-400';
    case 'requires_reconnect':
      return 'text-amber-500 dark:text-amber-400';
    case 'paused':
      return 'text-muted-foreground';
    case 'error':
      return 'text-destructive';
    default:
      return 'text-muted-foreground';
  }
}
