import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { HlmToaster } from '@spartan-ng/helm/sonner';
import { SystemNoticesHostComponent } from '@bowerbird/system-notices/angular';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, HlmToaster, SystemNoticesHostComponent],
  template: `
    <router-outlet></router-outlet>
    <hlm-toaster richColors closeButton position="bottom-right" />
    <bb-system-notices-host scope="global" />
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppComponent {}
