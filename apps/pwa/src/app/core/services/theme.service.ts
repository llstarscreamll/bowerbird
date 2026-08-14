import { Injectable, signal } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  readonly isDark = signal(typeof document !== 'undefined' && document.documentElement.classList.contains('dark'));

  constructor() {
    if (typeof document === 'undefined') {
      return;
    }

    const observer = new MutationObserver(() => {
      this.isDark.set(document.documentElement.classList.contains('dark'));
    });

    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    });
  }
}
