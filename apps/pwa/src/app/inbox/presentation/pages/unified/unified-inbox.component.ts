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
  host: {
    class: 'flex-1 flex flex-col min-h-0 w-full',
  },
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
    <div class="flex h-full w-full bg-white dark:bg-slate-950 text-slate-900 dark:text-white transition-colors duration-200">
      <!-- Master List (Left Pane) -->
      <aside class="w-[380px] flex flex-col border-r border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shrink-0 transition-colors duration-200">
        <!-- Top Toolbar & Search -->
        <div class="p-4 space-y-4 border-b border-slate-100 dark:border-slate-800 transition-colors">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-semibold flex items-center gap-2 text-slate-900 dark:text-white">
              <span class="material-icons-outlined text-slate-400 dark:text-slate-500 text-[20px]">inbox</span>
              Inbox
            </h2>
            <div class="flex items-center text-xs text-slate-500 dark:text-slate-400 gap-1 cursor-pointer hover:text-slate-800 dark:hover:text-slate-200 transition-colors">
              <span class="material-icons-outlined text-[16px]">check</span> Select
            </div>
          </div>

          <!-- Search Bar -->
          <div class="relative">
            <span class="material-icons-outlined absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 dark:text-slate-500 text-[18px]">search</span>
            <input
              type="text"
              placeholder="Search..."
              class="w-full pl-9 pr-8 py-2 bg-slate-50 dark:bg-slate-800 border-none rounded-lg text-sm text-slate-900 dark:text-white focus:ring-1 focus:ring-slate-200 dark:focus:ring-slate-700 placeholder:text-slate-400 dark:placeholder:text-slate-500 transition-shadow transition-colors"
              [ngModel]="filters().search"
              (ngModelChange)="setSearchFilter($event)"
            />
            <span
              class="absolute right-3 top-1/2 -translate-y-1/2 text-[10px] font-medium text-slate-400 dark:text-slate-500 border border-slate-200 dark:border-slate-700 rounded px-1.5 py-0.5 bg-white dark:bg-slate-900 transition-colors"
              >⌘K</span
            >
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
              class="px-3 py-1.5 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-300 text-sm font-medium rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors flex items-center gap-1.5"
              [class.bg-slate-100]="filters().onlyInvoices"
              [class.dark:bg-slate-700]="filters().onlyInvoices"
              (click)="setOnlyInvoicesFilter(!filters().onlyInvoices)"
            >
              <span class="material-icons-outlined text-[16px]">receipt_long</span>
            </button>
            <button
              class="px-3 py-1.5 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-300 text-sm font-medium rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors flex items-center gap-1.5"
            >
              <span class="material-icons-outlined text-[16px]">person</span>
            </button>
            <button
              class="px-3 py-1.5 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-300 text-sm font-medium rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors flex items-center gap-1.5"
            >
              <span class="material-icons-outlined text-[16px]">notifications</span>
            </button>
          </div>
        </div>

        <!-- Scrollable List -->
        <div class="flex-1 overflow-y-auto relative">
          <div *ngIf="error()" class="p-4 sticky top-0 z-10 bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm transition-colors">
            <app-alert type="error" title="Error de conexión" [message]="error() || ''" [dismissible]="true" (dismissed)="clearError()"> </app-alert>
          </div>

          <div *ngIf="loading()" class="p-8 text-center text-sm text-slate-500 dark:text-slate-400 transition-colors">Cargando mensajes...</div>

          <div *ngIf="!loading() && filteredMessages().length === 0" class="p-8 text-center">
            <p class="text-sm font-medium text-slate-700 dark:text-slate-300 transition-colors">No hay mensajes.</p>
          </div>

          <div *ngIf="!loading() && filteredMessages().length > 0">
            <div class="px-4 py-3 text-xs font-medium text-slate-500 dark:text-slate-400 flex items-center justify-between transition-colors">
              <span>Primary [{{ filteredMessages().length }}]</span>
            </div>

            <ul class="divide-y divide-slate-50 dark:divide-slate-800 transition-colors">
              <li
                *ngFor="let message of filteredMessages()"
                (click)="selectedMessage.set(message)"
                class="flex items-start gap-3 p-4 cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors relative"
                [class.bg-indigo-50]="selectedMessage()?.id === message.id"
                [class.dark:bg-indigo-500]="false"
                [class.dark:bg-opacity-10]="selectedMessage()?.id === message.id"
                [class.dark:bg-indigo-500/10]="selectedMessage()?.id === message.id"
                [class.hover:bg-indigo-50]="selectedMessage()?.id === message.id"
                [class.dark:hover:bg-indigo-500/10]="selectedMessage()?.id === message.id"
              >
                <!-- Active Indicator -->
                <div *ngIf="selectedMessage()?.id === message.id" class="absolute left-0 top-0 bottom-0 w-0.5 bg-indigo-600 dark:bg-indigo-500 transition-colors"></div>

                <!-- Avatar -->
                <div
                  class="w-10 h-10 rounded-full bg-slate-100 dark:bg-slate-800 flex items-center justify-center shrink-0 text-slate-600 dark:text-slate-300 font-medium text-sm overflow-hidden relative transition-colors"
                >
                  <ng-container *ngIf="message.provider === 'gmail'; else textAvatar">
                    <img src="https://www.gstatic.com/images/branding/product/1x/gmail_32dp.png" alt="Gmail" class="w-5 h-5 opacity-80" />
                  </ng-container>
                  <ng-template #textAvatar>
                    {{ message.sender.charAt(0) | uppercase }}
                  </ng-template>
                  <!-- Notification dot mockup -->
                  <div
                    *ngIf="message.processing_status === 'new'"
                    class="absolute bottom-0 right-0 w-2.5 h-2.5 bg-indigo-500 border-2 border-white dark:border-slate-900 rounded-full transition-colors"
                  ></div>
                </div>

                <!-- Content -->
                <div class="flex-1 min-w-0">
                  <div class="flex justify-between items-baseline mb-0.5">
                    <span class="font-semibold text-slate-900 dark:text-slate-100 text-sm truncate pr-2 transition-colors">{{ message.sender }}</span>
                    <span class="text-[11px] text-slate-500 dark:text-slate-400 shrink-0 transition-colors">{{ message.received_at | date: 'MMM d' }}</span>
                  </div>
                  <p
                    class="text-sm text-slate-500 dark:text-slate-400 truncate transition-colors"
                    [class.text-slate-900]="message.processing_status === 'new'"
                    [class.dark:text-slate-200]="message.processing_status === 'new'"
                  >
                    {{ message.subject || '(Sin asunto)' }}
                  </p>

                  <div class="flex items-center justify-between mt-1.5 h-4">
                    <!-- Tags -->
                    <div class="flex items-center gap-1.5">
                      <span *ngIf="message.has_xml" class="text-[10px] font-medium px-1.5 py-0.5 bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 rounded transition-colors"
                        >XML</span
                      >
                      <span *ngIf="message.has_pdf" class="text-[10px] font-medium px-1.5 py-0.5 bg-rose-50 dark:bg-rose-500/10 text-rose-700 dark:text-rose-400 rounded transition-colors">PDF</span>
                    </div>
                    <!-- Right icons -->
                    <div class="flex items-center gap-1 text-slate-400 dark:text-slate-500 transition-colors">
                      <span *ngIf="message.processing_status === 'error'" class="material-icons-outlined text-[14px] text-amber-500 dark:text-amber-400">warning</span>
                      <span
                        class="material-icons-outlined text-[14px]"
                        [class.text-indigo-500]="message.processing_status === 'processed'"
                        [class.dark:text-indigo-400]="message.processing_status === 'processed'"
                        >person</span
                      >
                    </div>
                  </div>
                </div>
              </li>
            </ul>
          </div>
        </div>
      </aside>

      <!-- Detail Pane (Right) -->
      <main class="flex-1 flex flex-col bg-white dark:bg-slate-950 overflow-hidden relative transition-colors duration-200">
        <ng-container *ngIf="selectedMessage(); else emptyState">
          <!-- Detail View -->
          <div class="flex-1 overflow-y-auto">
            <!-- Action Bar -->
            <div class="h-14 border-b border-slate-100 dark:border-slate-800 flex items-center px-6 gap-4 sticky top-0 bg-white dark:bg-slate-950 z-10 transition-colors">
              <button class="text-slate-400 dark:text-slate-500 hover:text-slate-600 dark:hover:text-slate-300 transition-colors" (click)="selectedMessage.set(null)">
                <span class="material-icons-outlined">arrow_back</span>
              </button>
              <div class="w-px h-4 bg-slate-200 dark:bg-slate-700 transition-colors"></div>
              <button class="text-slate-400 dark:text-slate-500 hover:text-slate-600 dark:hover:text-slate-300 transition-colors" title="Marcar como procesado">
                <span class="material-icons-outlined text-[20px]">check_circle</span>
              </button>
              <button class="text-slate-400 dark:text-slate-500 hover:text-slate-600 dark:hover:text-slate-300 transition-colors" title="Descargar adjuntos">
                <span class="material-icons-outlined text-[20px]">file_download</span>
              </button>
            </div>

            <!-- Content -->
            <div class="max-w-3xl mx-auto px-8 py-10">
              <h1 class="text-2xl font-semibold text-slate-900 dark:text-white mb-8 transition-colors">{{ selectedMessage()!.subject || '(Sin asunto)' }}</h1>

              <div class="flex items-start justify-between mb-8">
                <div class="flex items-center gap-3">
                  <div class="w-10 h-10 rounded-full bg-slate-100 dark:bg-slate-800 flex items-center justify-center text-slate-600 dark:text-slate-300 font-medium shrink-0 transition-colors">
                    {{ selectedMessage()!.sender.charAt(0) | uppercase }}
                  </div>
                  <div>
                    <div class="font-medium text-slate-900 dark:text-white transition-colors">{{ selectedMessage()!.sender }}</div>
                    <div class="text-xs text-slate-500 dark:text-slate-400 flex items-center gap-1 transition-colors">
                      Para: <span class="font-medium">{{ selectedMessage()!.account_email }}</span>
                    </div>
                  </div>
                </div>
                <div class="text-sm text-slate-500 dark:text-slate-400 flex items-center gap-2 transition-colors">
                  {{ selectedMessage()!.received_at | date: 'medium' }}
                  <span class="material-icons-outlined text-[18px]">star_border</span>
                </div>
              </div>

              <div class="prose prose-sm prose-slate dark:prose-invert max-w-none whitespace-pre-wrap text-slate-700 dark:text-slate-300 transition-colors">
                {{ selectedMessage()!.snippet || 'Este mensaje no contiene texto plano o aún no se ha extraído.' }}
              </div>

              <!-- Attachments Box (Mockup) -->
              <div
                *ngIf="selectedMessage()!.has_xml || selectedMessage()!.has_pdf"
                class="mt-10 p-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-900/50 transition-colors"
              >
                <h3 class="text-sm font-semibold text-slate-900 dark:text-white mb-3 transition-colors">Adjuntos</h3>
                <div class="flex flex-wrap gap-3">
                  <div
                    *ngIf="selectedMessage()!.has_xml"
                    class="flex items-center gap-3 p-3 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg w-64 shadow-sm hover:border-slate-300 dark:hover:border-slate-600 cursor-pointer transition-colors"
                  >
                    <div class="w-10 h-10 rounded bg-emerald-50 dark:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 flex items-center justify-center shrink-0 transition-colors">
                      <span class="material-icons-outlined text-[20px]">code</span>
                    </div>
                    <div class="min-w-0">
                      <p class="text-sm font-medium text-slate-900 dark:text-white truncate transition-colors">factura.xml</p>
                      <p class="text-xs text-slate-500 dark:text-slate-400 transition-colors">Documento electrónico</p>
                    </div>
                  </div>
                  <div
                    *ngIf="selectedMessage()!.has_pdf"
                    class="flex items-center gap-3 p-3 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg w-64 shadow-sm hover:border-slate-300 dark:hover:border-slate-600 cursor-pointer transition-colors"
                  >
                    <div class="w-10 h-10 rounded bg-rose-50 dark:bg-rose-500/10 text-rose-600 dark:text-rose-400 flex items-center justify-center shrink-0 transition-colors">
                      <span class="material-icons-outlined text-[20px]">picture_as_pdf</span>
                    </div>
                    <div class="min-w-0">
                      <p class="text-sm font-medium text-slate-900 dark:text-white truncate transition-colors">representacion.pdf</p>
                      <p class="text-xs text-slate-500 dark:text-slate-400 transition-colors">Representación gráfica</p>
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
              <div class="w-32 h-32 rounded-full bg-slate-50 dark:bg-slate-900 mx-auto mb-6 flex items-center justify-center relative shadow-[inset_0_2px_4px_rgba(0,0,0,0.02)] transition-colors">
                <!-- Inner card mockup -->
                <div
                  class="w-16 h-20 bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-100 dark:border-slate-700 p-2.5 flex flex-col gap-2 -rotate-6 transform hover:rotate-0 transition-all duration-300 cursor-default"
                >
                  <div class="flex items-center gap-1 border-b border-slate-100 dark:border-slate-700 pb-2 transition-colors">
                    <span class="material-icons-outlined text-[10px] text-slate-400 dark:text-slate-500">mail</span>
                    <div class="h-1.5 w-6 bg-slate-100 dark:bg-slate-700 rounded-full transition-colors"></div>
                  </div>
                  <div class="h-1.5 w-full bg-slate-100 dark:bg-slate-700 rounded-full transition-colors"></div>
                  <div class="h-1.5 w-2/3 bg-slate-100 dark:bg-slate-700 rounded-full transition-colors"></div>
                </div>
              </div>
              <h3 class="text-lg font-medium text-slate-900 dark:text-white transition-colors">It's empty here</h3>
              <p class="mt-1.5 text-sm text-slate-500 dark:text-slate-400 transition-colors">Choose an email to view details</p>
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
