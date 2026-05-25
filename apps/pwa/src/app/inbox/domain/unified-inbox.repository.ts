import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';
import { AccountHealthSummary, UnifiedInboxMessage } from './unified-inbox.model';

export interface UnifiedInboxRepository {
  listMessages(): Observable<UnifiedInboxMessage[]>;
  listAccountHealth(): Observable<AccountHealthSummary[]>;
}

export const UNIFIED_INBOX_REPOSITORY = new InjectionToken<UnifiedInboxRepository>('UNIFIED_INBOX_REPOSITORY');
