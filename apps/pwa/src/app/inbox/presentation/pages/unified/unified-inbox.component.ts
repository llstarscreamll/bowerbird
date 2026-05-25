import { CommonModule } from '@angular/common';
import { Component, OnDestroy, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { UnifiedInboxStore } from '../../../application/unified-inbox.store';
import { MessageProcessingStatus } from '../../../domain/unified-inbox.model';
import { MailProvider } from '../../../domain/inbox.types';
import { AccountStatusChipComponent } from '../../components/account-status-chip/account-status-chip.component';

@Component({
  selector: 'app-unified-inbox',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, AccountStatusChipComponent],
  styles: `
    .message-card-container {
      container-type: inline-size;
    }

    .message-card {
      display: grid;
      gap: 0.75rem;
      grid-template-columns: 1fr;
    }

    @container (min-width: 640px) {
      .message-card {
        align-items: center;
        grid-template-columns: minmax(0, 1fr) auto;
      }
    }
  `,
  template: `
    <div class="min-h-screen bg-slate-50 px-4 py-6 sm:px-6 lg:px-8">
      <div class="mx-auto grid max-w-7xl gap-6 lg:grid-cols-[320px_minmax(0,1fr)]">
        <aside class="card space-y-5 self-start">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">{{ tenantId() || 'tenant' }}</p>
            <h1 class="mt-2 text-xl font-semibold text-slate-900">Bandeja unificada</h1>
            <p class="mt-1 text-sm text-slate-600">Mensajes sincronizados de todos los proveedores conectados.</p>
          </div>

          <div class="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-3">
            <h2 class="text-xs font-semibold uppercase tracking-wide text-slate-600">Filtros</h2>

            <div>
              <label class="mb-1 block text-xs font-medium text-slate-600" for="provider">Proveedor</label>
              <select
                id="provider"
                class="input-field"
                [ngModel]="filters().provider"
                (ngModelChange)="setProviderFilter($event)"
              >
                <option value="all">Todos</option>
                <option *ngFor="let provider of providers" [value]="provider">{{ providerLabel(provider) }}</option>
              </select>
            </div>

            <div>
              <label class="mb-1 block text-xs font-medium text-slate-600" for="status">Estado</label>
              <select
                id="status"
                class="input-field"
                [ngModel]="filters().status"
                (ngModelChange)="setStatusFilter($event)"
              >
                <option value="all">Todos</option>
                <option *ngFor="let status of statuses" [value]="status">{{ messageStatusLabel(status) }}</option>
              </select>
            </div>

            <div>
              <label class="mb-1 block text-xs font-medium text-slate-600" for="search">Buscar</label>
              <input
                id="search"
                class="input-field"
                [ngModel]="filters().search"
                (ngModelChange)="setSearchFilter($event)"
                placeholder="Remitente, asunto o cuenta"
              />
            </div>

            <label class="flex items-center gap-2 text-xs text-slate-700">
              <input
                type="checkbox"
                [ngModel]="filters().onlyInvoices"
                (ngModelChange)="setOnlyInvoicesFilter($event)"
                class="h-4 w-4 rounded border-slate-300"
              />
              Solo mensajes con XML o PDF
            </label>
          </div>

          <div class="space-y-2 rounded-xl border border-slate-200 bg-white p-3">
            <h2 class="text-xs font-semibold uppercase tracking-wide text-slate-600">Estado de cuentas</h2>
            <ul class="space-y-2" *ngIf="accountHealth().length > 0; else noAccounts">
              <li *ngFor="let item of accountHealth()" class="rounded-lg border border-slate-200 p-2">
                <p class="truncate text-xs font-medium text-slate-900">{{ item.email_address }}</p>
                <div class="mt-1 flex items-center justify-between gap-2 text-[11px]">
                  <span class="text-slate-500">{{ providerLabel(item.provider) }}</span>
                  <app-account-status-chip [status]="item.status" />
                </div>
              </li>
            </ul>
            <ng-template #noAccounts>
              <p class="text-xs text-slate-500">No hay cuentas conectadas.</p>
            </ng-template>
          </div>

          <a
            [routerLink]="['/', tenantId(), 'inbox', 'connections']"
            class="inline-flex text-xs font-medium text-indigo-600 hover:text-indigo-500"
          >
            Gestionar conexiones de correo
          </a>
        </aside>

        <section class="card p-0">
          <div class="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200 px-5 py-4">
            <h2 class="text-sm font-semibold uppercase tracking-wide text-slate-600">Mensajes</h2>
            <span class="rounded-full border border-slate-200 bg-slate-50 px-2 py-1 text-xs text-slate-600">
              {{ filteredMessages().length }} resultados
            </span>
          </div>

          <div *ngIf="loading()" class="px-5 py-10 text-sm text-slate-500">Cargando mensajes...</div>

          <div *ngIf="!loading() && filteredMessages().length === 0" class="px-5 py-10 text-center">
            <p class="text-sm font-medium text-slate-700">No hay mensajes para estos filtros.</p>
            <p class="mt-1 text-sm text-slate-500">Ajusta los filtros o espera una nueva sincronizacion.</p>
          </div>

          <ul *ngIf="!loading() && filteredMessages().length > 0" class="divide-y divide-slate-200">
            <li *ngFor="let message of filteredMessages()" class="message-card-container px-5 py-4 hover:bg-slate-50">
              <article class="message-card">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <span
                      class="rounded-full border px-2 py-0.5 text-[11px]"
                      [ngClass]="providerClasses(message.provider)"
                    >
                      {{ providerLabel(message.provider) }}
                    </span>
                    <span
                      class="rounded-full border px-2 py-0.5 text-[11px]"
                      [ngClass]="messageStatusClasses(message.processing_status)"
                    >
                      {{ messageStatusLabel(message.processing_status) }}
                    </span>
                    <span class="text-xs text-slate-500">{{ message.account_email }}</span>
                  </div>

                  <h3 class="mt-1 truncate text-sm font-semibold text-slate-900">
                    {{ message.subject || '(Sin asunto)' }}
                  </h3>
                  <p class="truncate text-sm text-slate-600">{{ message.sender }}</p>
                  <p class="mt-1 text-xs text-slate-500" *ngIf="message.snippet">{{ message.snippet }}</p>
                </div>

                <div class="flex flex-wrap items-center gap-2 text-xs text-slate-500">
                  <span class="rounded-full border border-slate-200 bg-white px-2 py-1">{{
                    message.received_at | date: 'short'
                  }}</span>
                  <span class="rounded-full border border-slate-200 bg-white px-2 py-1" *ngIf="message.has_xml"
                    >XML</span
                  >
                  <span class="rounded-full border border-slate-200 bg-white px-2 py-1" *ngIf="message.has_pdf"
                    >PDF</span
                  >
                </div>
              </article>
            </li>
          </ul>
        </section>
      </div>
    </div>
  `,
})
export class UnifiedInboxComponent implements OnInit, OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly store = inject(UnifiedInboxStore);

  readonly loading = this.store.loading;
  readonly tenantId = signal('');
  readonly messages = this.store.messages;
  readonly accountHealth = this.store.accountHealth;
  readonly filteredMessages = this.store.filteredMessages;
  readonly filters = this.store.filters;

  readonly providers = this.store.providers;
  readonly statuses = this.store.statuses;

  ngOnInit(): void {
    this.tenantId.set(this.route.snapshot.paramMap.get('tenantId') || '');
    this.store.init();
  }

  ngOnDestroy(): void {
    this.store.destroy();
  }

  providerLabel(provider: MailProvider): string {
    return this.store.providerLabel(provider);
  }

  messageStatusLabel(status: MessageProcessingStatus): string {
    return this.store.messageStatusLabel(status);
  }

  messageStatusClasses(status: MessageProcessingStatus): string {
    return this.store.messageStatusClasses(status);
  }

  providerClasses(provider: MailProvider): string {
    return this.store.providerClasses(provider);
  }

  setProviderFilter(provider: 'all' | MailProvider): void {
    this.store.patchFilters({ provider });
  }

  setStatusFilter(status: 'all' | MessageProcessingStatus): void {
    this.store.patchFilters({ status });
  }

  setSearchFilter(search: string): void {
    this.store.patchFilters({ search });
  }

  setOnlyInvoicesFilter(onlyInvoices: boolean): void {
    this.store.patchFilters({ onlyInvoices });
  }
}
