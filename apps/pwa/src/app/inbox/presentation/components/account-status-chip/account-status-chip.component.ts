import { CommonModule } from '@angular/common';
import { Component, Input } from '@angular/core';
import { ConnectionStatus } from '../../../domain/inbox.types';

@Component({
  selector: 'app-account-status-chip',
  standalone: true,
  imports: [CommonModule],
  template: `
    <span class="rounded-full border px-2 py-0.5" [ngClass]="classes()">
      {{ label() }}
    </span>
  `,
})
export class AccountStatusChipComponent {
  @Input({ required: true }) status!: ConnectionStatus;

  label(): string {
    switch (this.status) {
      case 'active':
        return 'Activa';
      case 'requires_reconnect':
        return 'Reconexion';
      case 'paused':
        return 'Pausada';
      case 'error':
        return 'Error';
      default:
        return this.status;
    }
  }

  classes(): string {
    switch (this.status) {
      case 'active':
        return 'border-emerald-200 bg-emerald-50 text-emerald-700';
      case 'requires_reconnect':
        return 'border-amber-200 bg-amber-50 text-amber-700';
      case 'paused':
        return 'border-slate-200 bg-slate-50 text-slate-700';
      case 'error':
        return 'border-rose-200 bg-rose-50 text-rose-700';
      default:
        return 'border-slate-200 bg-slate-50 text-slate-700';
    }
  }
}
