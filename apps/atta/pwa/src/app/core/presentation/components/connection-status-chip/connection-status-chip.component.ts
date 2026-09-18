import { Component, computed, input } from '@angular/core';
import { ConnectionStatus } from '../../../domain/connection-status.model';
import { connectionStatusBadgeLabel, connectionStatusBadgeVariant } from '../../connection-status';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';

@Component({
  selector: 'app-connection-status-chip',
  standalone: true,
  imports: [HlmBadgeImports],
  template: ` <span hlmBadge [variant]="badgeVariant()">{{ label() }}</span> `,
})
export class ConnectionStatusChipComponent {
  readonly status = input.required<ConnectionStatus>();

  readonly label = computed(() => connectionStatusBadgeLabel(this.status()));
  readonly badgeVariant = computed(() => connectionStatusBadgeVariant(this.status()));
}
