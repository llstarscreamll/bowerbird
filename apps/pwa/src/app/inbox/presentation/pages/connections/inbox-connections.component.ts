import { CommonModule } from '@angular/common';
import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { InboxConnectionsStore } from '../../../application/inbox-connections.store';
import { ConnectedAccount } from '../../../domain/inbox-connections.model';
import { ConnectionStatus, MailProvider } from '../../../domain/inbox.types';
import { AccountStatusChipComponent } from '../../components/account-status-chip/account-status-chip.component';
import { AlertComponent } from '../../../../core/presentation/components/alert/alert.component';

@Component({
  selector: 'app-inbox-connections',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, AccountStatusChipComponent, AlertComponent],
  host: {
    class: 'flex-1 flex flex-col min-h-0 w-full',
  },
  template: `
    <div class="h-full bg-gradient-to-b from-slate-50 to-white dark:from-slate-950 dark:to-slate-900 px-4 py-8 sm:px-6 lg:px-8 overflow-y-auto transition-colors duration-200">
      <div class="mx-auto w-full max-w-6xl space-y-6">
        <header class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm sm:p-8">
          <p class="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">Tenant: {{ tenantId() || 'N/A' }}</p>
          <h1 class="mt-2 text-2xl font-semibold tracking-tight text-slate-900 sm:text-3xl">Conexiones de correo</h1>
          <p class="mt-2 max-w-3xl text-sm text-slate-600">Conecta y gestiona cuentas de Gmail, Outlook y otros proveedores para sincronizar facturas electronicas.</p>
          <div class="mt-5 flex flex-wrap gap-2 text-xs">
            <span class="rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-emerald-700"> Activas: {{ statusCount('active') }} </span>
            <span class="rounded-full border border-amber-200 bg-amber-50 px-3 py-1 text-amber-700"> Reconexion: {{ statusCount('requires_reconnect') }} </span>
            <span class="rounded-full border border-rose-200 bg-rose-50 px-3 py-1 text-rose-700"> Errores: {{ statusCount('error') }} </span>
            <span class="rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-slate-700"> Pausadas: {{ statusCount('paused') }} </span>
          </div>
        </header>

        <section class="grid gap-6 lg:grid-cols-[1fr_320px]">
          <div class="card p-0">
            <div class="border-b border-slate-200 px-5 py-4">
              <h2 class="text-sm font-semibold uppercase tracking-wide text-slate-600">Cuentas conectadas</h2>
            </div>

            <div *ngIf="loading()" class="px-5 py-10 text-sm text-slate-500">Cargando cuentas...</div>

            <div *ngIf="!loading() && accounts().length === 0" class="px-5 py-10 text-center">
              <p class="text-sm font-medium text-slate-700">No hay cuentas conectadas todavia</p>
              <p class="mt-1 text-sm text-slate-500">Usa el panel de la derecha para conectar la primera cuenta.</p>
            </div>

            <ul *ngIf="!loading() && accounts().length > 0" class="divide-y divide-slate-200">
              <li *ngFor="let account of accounts()" class="px-5 py-4">
                <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div class="min-w-0">
                    <div class="flex items-center gap-2">
                      <p class="truncate text-sm font-semibold text-slate-900">{{ account.email_address }}</p>
                      <app-account-status-chip [status]="account.status" />
                    </div>
                    <p class="mt-1 text-xs text-slate-500">
                      {{ providerLabel(account.provider) }}
                      <span *ngIf="account.last_synced_at">- Ultima sincronizacion: {{ account.last_synced_at | date: 'short' }}</span>
                    </p>
                    <p *ngIf="account.last_error" class="mt-1 text-xs text-rose-600">{{ account.last_error }}</p>
                  </div>
                  <div class="flex gap-2">
                    <button class="btn-secondary px-3 py-2 text-xs" [disabled]="disconnectingId() === account.id" (click)="disconnect(account)">
                      {{ disconnectingId() === account.id ? 'Desconectando...' : 'Desconectar' }}
                    </button>
                  </div>
                </div>
              </li>
            </ul>
          </div>

          <aside class="card space-y-4">
            <div>
              <h3 class="text-sm font-semibold uppercase tracking-wide text-slate-600">Conectar cuenta</h3>
              <p class="mt-1 text-xs text-slate-500">Puedes conectar multiples cuentas del mismo proveedor.</p>
            </div>

            <form (ngSubmit)="connect()" class="space-y-4">
              <div>
                <label for="provider" class="mb-1 block text-xs font-medium text-slate-600">Proveedor</label>
                <select id="provider" name="provider" class="input-field" [(ngModel)]="newProvider" required>
                  <option *ngFor="let provider of providers" [value]="provider">{{ providerLabel(provider) }}</option>
                </select>
              </div>

              <div>
                <label for="email" class="mb-1 block text-xs font-medium text-slate-600">Correo de la cuenta</label>
                <input id="email" name="email" type="email" class="input-field" [(ngModel)]="newEmailAddress" required placeholder="facturas@empresa.com" />
              </div>

              <app-alert *ngIf="errorMessage()" type="error">
                {{ errorMessage() }}
              </app-alert>

              <button class="btn-primary w-full" type="submit" [disabled]="submitting()">
                {{ submitting() ? 'Conectando...' : 'Conectar cuenta' }}
              </button>
            </form>

            <a [routerLink]="['/', tenantId(), 'inbox', 'connections']" class="inline-flex text-xs font-medium text-indigo-600 hover:text-indigo-500"> Actualizar vista </a>
            <a [routerLink]="['/', tenantId(), 'inbox', 'unified']" class="inline-flex text-xs font-medium text-indigo-600 hover:text-indigo-500"> Ir a bandeja unificada </a>
          </aside>
        </section>
      </div>
    </div>
  `,
})
export class InboxConnectionsComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  readonly store = inject(InboxConnectionsStore);

  readonly providers = this.store.providers;
  readonly accounts = this.store.accounts;
  readonly loading = this.store.loading;
  readonly submitting = this.store.submitting;
  readonly disconnectingId = this.store.disconnectingId;
  readonly errorMessage = this.store.errorMessage;
  readonly tenantId = signal('');

  newProvider: MailProvider = 'gmail';
  newEmailAddress = '';

  ngOnInit(): void {
    this.tenantId.set(this.route.snapshot.paramMap.get('tenantId') || '');
    this.store.loadAccounts();
  }

  loadAccounts(): void {
    this.store.loadAccounts();
  }

  connect(): void {
    const email = this.newEmailAddress;
    this.store.connectAccount(this.newProvider, email, (authURL) => window.location.assign(authURL));
    this.newEmailAddress = '';
  }

  disconnect(account: ConnectedAccount): void {
    const confirmed = window.confirm(`Se desconectara ${account.email_address}. Deseas continuar?`);
    if (!confirmed) {
      return;
    }
    this.store.disconnectAccount(account.id);
  }

  providerLabel(provider: MailProvider): string {
    return this.store.providerLabel(provider);
  }

  statusCount(status: ConnectionStatus): number {
    return this.accounts().filter((account) => account.status === status).length;
  }
}
