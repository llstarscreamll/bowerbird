import { isPlatformBrowser } from '@angular/common';
import { Injectable, Inject, PLATFORM_ID } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class FlowbiteService {
  constructor(@Inject(PLATFORM_ID) private platformId: any) {}

  load(callback: (flowbite: any) => void) {
    if (!isPlatformBrowser(this.platformId)) {
        return;
    }

    import('flowbite').then(flowbite => callback(flowbite));
  }
}