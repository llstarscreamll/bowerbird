import { Injectable } from '@angular/core';
import type { ClockPort } from '../application/ports/clock.port';

@Injectable({ providedIn: 'root' })
export class SystemClockAdapter implements ClockPort {
  now(): Date {
    return new Date();
  }
}
