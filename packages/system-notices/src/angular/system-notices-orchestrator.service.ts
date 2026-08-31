import { Injectable, inject } from '@angular/core';
import { SystemNoticesOrchestrator } from '../lib/system-notices.orchestrator';
import { SYSTEM_NOTICE } from './system-notice.token';
import type { SystemNotice } from '../lib/system-notice.port';

@Injectable()
export class SystemNoticesOrchestratorService extends SystemNoticesOrchestrator {
  constructor() {
    const injected = inject(SYSTEM_NOTICE, { optional: true });
    const notices: SystemNotice[] = !injected ? [] : Array.isArray(injected) ? injected : [injected];
    super(notices);
  }
}
