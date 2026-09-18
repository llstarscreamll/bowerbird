import { DOCUMENT, isPlatformBrowser } from '@angular/common';
import { Injectable, PLATFORM_ID, inject } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class ViewportAdapter {
  private readonly document = inject(DOCUMENT);
  private readonly platformId = inject(PLATFORM_ID);

  private static readonly MOBILE_BREAKPOINT = '768px';

  isMobile(): boolean {
    if (!isPlatformBrowser(this.platformId)) {
      return false;
    }

    return this.document.defaultView?.matchMedia(`(max-width: ${ViewportAdapter.MOBILE_BREAKPOINT})`).matches ?? false;
  }
}
