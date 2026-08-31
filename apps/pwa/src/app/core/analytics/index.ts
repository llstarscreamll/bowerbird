import { EnvironmentProviders, makeEnvironmentProviders } from '@angular/core';
import { ANALYTICS_PORT } from './application/ports/analytics.port';
import { ConsoleAnalyticsAdapter } from './infrastructure/console-analytics.adapter';

export function provideAnalytics(): EnvironmentProviders {
  return makeEnvironmentProviders([{ provide: ANALYTICS_PORT, useClass: ConsoleAnalyticsAdapter }]);
}

export { ANALYTICS_PORT, type AnalyticsPort } from './application/ports/analytics.port';
