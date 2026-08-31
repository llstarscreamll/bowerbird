import { InjectionToken } from '@angular/core';

export interface ClockPort {
  now(): Date;
}

export const CLOCK_PORT = new InjectionToken<ClockPort>('CLOCK_PORT');
