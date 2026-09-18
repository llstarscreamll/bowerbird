import { ChangeDetectionStrategy, Component, inject, input, OnInit } from '@angular/core';
import type { SystemNoticeScope } from '../lib/system-notice.port';
import { SystemNoticesOrchestratorService } from './system-notices-orchestrator.service';

@Component({
  selector: 'bb-system-notices-host',
  standalone: true,
  template: '',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SystemNoticesHostComponent implements OnInit {
  readonly scope = input.required<SystemNoticeScope>();

  private readonly orchestrator = inject(SystemNoticesOrchestratorService);

  ngOnInit(): void {
    this.orchestrator.tryPresent(this.scope());
  }

  refresh(): void {
    this.orchestrator.tryPresent(this.scope());
  }
}
