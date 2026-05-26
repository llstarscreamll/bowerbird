import { CommonModule } from '@angular/common';
import { Component, OnDestroy, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { UnifiedInboxStore } from '../../../application/unified-inbox.store';
import { MessageProcessingStatus } from '../../../domain/unified-inbox.model';
import { MailProvider } from '../../../domain/inbox.types';
import { AlertComponent } from '../../../../core/presentation/components/alert/alert.component';

@Component({
  selector: 'app-unified-inbox',
  standalone: true,
  imports: [CommonModule, FormsModule, AlertComponent],
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
    <div class="flex h-full w-full bg-white text-slate-900">
      <!-- Master List (Left Pane) -->
      <aside class="w-[380px] flex flex-col border-r border-slate-200 bg-white shrink-0">
        <!-- Top Toolbar & Search -->
        <div class="p-4 space-y-4 border-b border-slate-100">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-semibold flex items-center gap-2">
              <span class="material-icons-outlined text-slate-400 text-[20px]">inbox</span>
              Inbox
            </h2>
            <div class="flex items-center text-xs text-slate-500 gap-1 cursor-pointer hover:text-slate-800"><span class="material-icons-outlined text-[16px]">check</span> Select</div>
          </div>

          <!-- Search Bar -->
          <div class="relative">
            <span class="material-icons-outlined absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 text-[18px]">search</span>
            <input
              type="text"
              placeholder="Search..."
              class="w-full pl-9 pr-8 py-2 bg-slate-50 border-none rounded-lg text-sm text-slate-900 focus:ring-1 focus:ring-slate-200 placeholder:text-slate-400 transition-shadow"
              [ngModel]="filters().search"
              (ngModelChange)="setSearchFilter($event)"
            />
            <span class="absolute right-3 top-1/2 -translate-y-1/2 text-[10px] font-medium text-slate-400 border border-slate-200 rounded px-1.5 py-0.5 bg-white">⌘K</span>
          </div>

          <!-- Quick Filters -->
          <div class="flex items-center gap-2">
            <button
              class="flex-1 px-3 py-1.5 bg-indigo-600 text-white text-sm font-medium rounded-lg flex items-center justify-center gap-1.5 hover:bg-indigo-700 transition-colors"
              (click)="setOnlyInvoicesFilter(false)"
            >
              <span class="material-icons-outlined text-[16px]">bolt</span> Primary
            </button>
            <button
              class="px-3 py-1.5 bg-white border border-slate-200 text-slate-600 text-sm font-medium rounded-lg hover:bg-slate-50 transition-colors flex items-center gap-1.5"
              [class.bg-slate-100]="filters().onlyInvoices"
              (click)="setOnlyInvoicesFilter(!filters().onlyInvoices)"
            >
              <span class="material-icons-outlined text-[16px]">receipt_long</span>
            </button>
            <button class="px-3 py-1.5 bg-white border border-slate-200 text-slate-600 text-sm font-medium rounded-lg hover:bg-slate-50 transition-colors flex items-center gap-1.5">
              <span class="material-icons-outlined text-[16px]">person</span>
            </button>
            <button class="px-3 py-1.5 bg-white border border-slate-200 text-slate-600 text-sm font-medium rounded-lg hover:bg-slate-50 transition-colors flex items-center gap-1.5">
              <span class="material-icons-outlined text-[16px]">notifications</span>
            </button>
          </div>
        </div>

        <!-- Scrollable List -->
        <div class="flex-1 overflow-y-auto relative">
          <div *ngIf="error()" class="p-4 sticky top-0 z-10 bg-white/80 backdrop-blur-sm">
            <app-alert type="error" title="Error de conexión" [message]="error() || ''" [dismissible]="true" (dismissed)="clearError()"> </app-alert>
          </div>

          <div *ngIf="loading()" class="p-8 text-center text-sm text-slate-500">Cargando mensajes...</div>

          <div *ngIf="!loading() && filteredMessages().length === 0" class="p-8 text-center">
            <p class="text-sm font-medium text-slate-700">No hay mensajes.</p>
          </div>

          <div *ngIf="!loading() && filteredMessages().length > 0">
            <div class="px-4 py-3 text-xs font-medium text-slate-500 flex items-center justify-between">
              <span>Primary [{{ filteredMessages().length }}]</span>
            </div>

            <ul class="divide-y divide-slate-50">
              <li
                *ngFor="let message of filteredMessages()"
                (click)="selectedMessage.set(message)"
                class="flex items-start gap-3 p-4 cursor-pointer hover:bg-slate-50 transition-colors relative"
                [class.bg-indigo-50]="selectedMessage()?.id === message.id"
                [class.hover:bg-indigo-50]="selectedMessage()?.id === message.id"
              >
                <!-- Active Indicator -->
                <div *ngIf="selectedMessage()?.id === message.id" class="absolute left-0 top-0 bottom-0 w-0.5 bg-indigo-600"></div>

                <!-- Avatar -->
                <div class="w-10 h-10 rounded-full bg-slate-100 flex items-center justify-center shrink-0 text-slate-600 font-medium text-sm overflow-hidden relative">
                  <ng-container *ngIf="message.provider === 'gmail'; else textAvatar">
                    <img src="https://www.gstatic.com/images/branding/product/1x/gmail_32dp.png" alt="Gmail" class="w-5 h-5 opacity-80" />
                  </ng-container>
                  <ng-template #textAvatar>
                    {{ message.sender.charAt(0) | uppercase }}
                  </ng-template>
                  <!-- Notification dot mockup -->
                  <div *ngIf="message.processing_status === 'new'" class="absolute bottom-0 right-0 w-2.5 h-2.5 bg-indigo-500 border-2 border-white rounded-full"></div>
                </div>

                <!-- Content -->
                <div class="flex-1 min-w-0">
                  <div class="flex justify-between items-baseline mb-0.5">
                    <span class="font-semibold text-slate-900 text-sm truncate pr-2">{{ message.sender }}</span>
                    <span class="text-[11px] text-slate-500 shrink-0">{{ message.received_at | date: 'MMM d' }}</span>
                  </div>
                  <p class="text-sm text-slate-500 truncate" [class.text-slate-900]="message.processing_status === 'new'">
                    {{ message.subject || '(Sin asunto)' }}
                  </p>

                  <div class="flex items-center justify-between mt-1.5 h-4">
                    <!-- Tags -->
                    <div class="flex items-center gap-1.5">
                      <span *ngIf="message.has_xml" class="text-[10px] font-medium px-1.5 py-0.5 bg-emerald-50 text-emerald-700 rounded">XML</span>
                      <span *ngIf="message.has_pdf" class="text-[10px] font-medium px-1.5 py-0.5 bg-rose-50 text-rose-700 rounded">PDF</span>
                    </div>
                    <!-- Right icons -->
                    <div class="flex items-center gap-1 text-slate-400">
                      <span *ngIf="message.processing_status === 'error'" class="material-icons-outlined text-[14px] text-amber-500">warning</span>
                      <span class="material-icons-outlined text-[14px]" [class.text-indigo-500]="message.processing_status === 'processed'">person</span>
                    </div>
                  </div>
                </div>
              </li>
            </ul>
          </div>
        </div>
      </aside>

      <!-- Detail Pane (Right) -->
      <main class="flex-1 flex flex-col bg-white overflow-hidden relative">
        <ng-container *ngIf="selectedMessage(); else emptyState">
          <!-- Detail View -->
          <div class="flex-1 overflow-y-auto">
            <!-- Action Bar -->
            <div class="h-14 border-b border-slate-100 flex items-center px-6 gap-4 sticky top-0 bg-white z-10">
              <button class="text-slate-400 hover:text-slate-600 transition-colors" (click)="selectedMessage.set(null)">
                <span class="material-icons-outlined">arrow_back</span>
              </button>
              <div class="w-px h-4 bg-slate-200"></div>
              <button class="text-slate-400 hover:text-slate-600 transition-colors" title="Marcar como procesado">
                <span class="material-icons-outlined text-[20px]">check_circle</span>
              </button>
              <button class="text-slate-400 hover:text-slate-600 transition-colors" title="Descargar adjuntos">
                <span class="material-icons-outlined text-[20px]">file_download</span>
              </button>
            </div>

            <!-- Content -->
            <div class="max-w-3xl mx-auto px-8 py-10">
              <h1 class="text-2xl font-semibold text-slate-900 mb-8">{{ selectedMessage()!.subject || '(Sin asunto)' }}</h1>

              <div class="flex items-start justify-between mb-8">
                <div class="flex items-center gap-3">
                  <div class="w-10 h-10 rounded-full bg-slate-100 flex items-center justify-center text-slate-600 font-medium shrink-0">
                    {{ selectedMessage()!.sender.charAt(0) | uppercase }}
                  </div>
                  <div>
                    <div class="font-medium text-slate-900">{{ selectedMessage()!.sender }}</div>
                    <div class="text-xs text-slate-500 flex items-center gap-1">
                      Para: <span class="font-medium">{{ selectedMessage()!.account_email }}</span>
                    </div>
                  </div>
                </div>
                <div class="text-sm text-slate-500 flex items-center gap-2">
                  {{ selectedMessage()!.received_at | date: 'medium' }}
                  <span class="material-icons-outlined text-[18px]">star_border</span>
                </div>
              </div>

              <div class="prose prose-sm prose-slate max-w-none whitespace-pre-wrap text-slate-700">
                {{ selectedMessage()!.snippet || 'Este mensaje no contiene texto plano o aún no se ha extraído.' }}
              </div>

              <!-- Attachments Box (Mockup) -->
              <div *ngIf="selectedMessage()!.has_xml || selectedMessage()!.has_pdf" class="mt-10 p-4 rounded-xl border border-slate-200 bg-slate-50">
                <h3 class="text-sm font-semibold text-slate-900 mb-3">Adjuntos</h3>
                <div class="flex flex-wrap gap-3">
                  <div
                    *ngIf="selectedMessage()!.has_xml"
                    class="flex items-center gap-3 p-3 bg-white border border-slate-200 rounded-lg w-64 shadow-sm hover:border-slate-300 cursor-pointer transition-colors"
                  >
                    <div class="w-10 h-10 rounded bg-emerald-50 text-emerald-600 flex items-center justify-center shrink-0">
                      <span class="material-icons-outlined text-[20px]">code</span>
                    </div>
                    <div class="min-w-0">
                      <p class="text-sm font-medium text-slate-900 truncate">factura.xml</p>
                      <p class="text-xs text-slate-500">Documento electrónico</p>
                    </div>
                  </div>
                  <div
                    *ngIf="selectedMessage()!.has_pdf"
                    class="flex items-center gap-3 p-3 bg-white border border-slate-200 rounded-lg w-64 shadow-sm hover:border-slate-300 cursor-pointer transition-colors"
                  >
                    <div class="w-10 h-10 rounded bg-rose-50 text-rose-600 flex items-center justify-center shrink-0">
                      <span class="material-icons-outlined text-[20px]">picture_as_pdf</span>
                    </div>
                    <div class="min-w-0">
                      <p class="text-sm font-medium text-slate-900 truncate">representacion.pdf</p>
                      <p class="text-xs text-slate-500">Representación gráfica</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </ng-container>

        <!-- Empty State -->
        <ng-template #emptyState>
          <div class="absolute inset-0 flex items-center justify-center p-8">
            <div class="text-center max-w-sm mx-auto">
              <!-- Illustration Circle -->
              <div class="w-32 h-32 rounded-full bg-slate-50 mx-auto mb-6 flex items-center justify-center relative shadow-[inset_0_2px_4px_rgba(0,0,0,0.02)]">
                <!-- Inner card mockup -->
                <div
                  class="w-16 h-20 bg-white rounded-lg shadow-sm border border-slate-100 p-2.5 flex flex-col gap-2 -rotate-6 transform hover:rotate-0 transition-transform duration-300 cursor-default"
                >
                  <div class="flex items-center gap-1 border-b border-slate-100 pb-2">
                    <span class="material-icons-outlined text-[10px] text-slate-400">mail</span>
                    <div class="h-1.5 w-6 bg-slate-100 rounded-full"></div>
                  </div>
                  <div class="h-1.5 w-full bg-slate-100 rounded-full"></div>
                  <div class="h-1.5 w-2/3 bg-slate-100 rounded-full"></div>
                </div>
              </div>
              <h3 class="text-lg font-medium text-slate-900">It's empty here</h3>
              <p class="mt-1.5 text-sm text-slate-500">Choose an email to view details</p>
            </div>
          </div>
        </ng-template>
      </main>
    </div>
  `,
})
export class UnifiedInboxComponent implements OnInit, OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly store = inject(UnifiedInboxStore);

  readonly loading = this.store.loading;
  readonly error = this.store.error;
  readonly tenantId = signal('');
  readonly messages = this.store.messages;
  readonly accountHealth = this.store.accountHealth;
  readonly filteredMessages = this.store.filteredMessages;
  readonly filters = this.store.filters;

  readonly providers = this.store.providers;
  readonly statuses = this.store.statuses;

  readonly selectedMessage = signal<any | null>(null);

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

  clearError(): void {
    this.store.clearError();
  }
}
