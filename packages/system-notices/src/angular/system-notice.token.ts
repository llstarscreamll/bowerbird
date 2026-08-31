import { InjectionToken } from '@angular/core';
import type { SystemNotice } from '../lib/system-notice.port';

export const SYSTEM_NOTICE = new InjectionToken<SystemNotice>('SYSTEM_NOTICE');
