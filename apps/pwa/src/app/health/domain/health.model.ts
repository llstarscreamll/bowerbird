export type HealthStatus = 'checking...' | 'ok' | 'degraded';

export interface HealthInfo {
  status: HealthStatus;
  lastChecked: Date | null;
}
