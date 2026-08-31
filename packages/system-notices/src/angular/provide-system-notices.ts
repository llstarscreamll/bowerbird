import { EnvironmentProviders, makeEnvironmentProviders } from '@angular/core';
import { SystemNoticesOrchestratorService } from './system-notices-orchestrator.service';

export function provideSystemNotices(): EnvironmentProviders {
  return makeEnvironmentProviders([SystemNoticesOrchestratorService]);
}
