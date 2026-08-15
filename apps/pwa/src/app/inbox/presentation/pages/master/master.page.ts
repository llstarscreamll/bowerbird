import { DatePipe, NgClass, UpperCasePipe } from '@angular/common';
import { Component, OnDestroy, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmDropdownMenuImports } from '@spartan-ng/helm/dropdown-menu';
import { HlmInputImports } from '@spartan-ng/helm/input';
import { HlmSeparatorImports } from '@spartan-ng/helm/separator';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { UnifiedInboxStore } from '../../../application/unified-inbox.store';
import { EntitlementsStore } from '../../../../entitlements/application/entitlements.store';
import { AccountHealthSummary, MailFolder, UnifiedInboxMessage } from '../../../domain/unified-inbox.model';
import { MailProvider, providerLabel } from '../../../domain/inbox.types';
import { connectionStatusDetailLabel, connectionStatusIcon, connectionStatusIconClasses } from '../../../../core/presentation/connection-status';
import { resolveConnectionStatus } from '../../../../core/domain/connection-status.model';
import { SecureEmailBodyComponent } from '../../components/secure-email-body/secure-email-body.component';

@Component({
  selector: 'app-master-inbox',
  standalone: true,
  imports: [
    FormsModule,
    NgIcon,
    NgClass,
    DatePipe,
    UpperCasePipe,
    HlmAlertImports,
    HlmBadgeImports,
    HlmButtonImports,
    HlmCardImports,
    HlmDropdownMenuImports,
    HlmInputImports,
    HlmSeparatorImports,
    HlmSpinnerImports,
    SecureEmailBodyComponent,
  ],
  host: {
    class: 'flex-1 flex flex-col min-h-0 w-full',
  },
  template: `
    <div class="relative flex h-full w-full bg-background text-foreground">
      <aside class="flex w-[380px] shrink-0 flex-col border-r border-border bg-card">
        <div class="space-y-4 border-b border-border p-4">
          <div class="flex items-center justify-between">
            <h2 class="flex items-center gap-2 text-lg font-semibold">
              <ng-icon name="lucideInbox" class="text-[20px] text-muted-foreground" />
              Inbox
            </h2>
            @if (accountHealth().length > 0) {
              <button type="button" hlmBtn variant="outline" size="sm" class="max-w-[160px] justify-between gap-1.5" [hlmDropdownMenuTrigger]="accountMenu">
                <span class="flex-1 truncate">{{ currentAccountLabel() }}</span>
                @if (selectedAccount(); as account) {
                  <ng-icon
                    [name]="connectionStatusIcon(account)"
                    class="text-[15px]"
                    [ngClass]="connectionStatusIconClasses(account)"
                    [title]="connectionStatusLabel(account)"
                    [attr.aria-label]="connectionStatusLabel(account)"
                  />
                } @else {
                  <ng-icon name="lucideInbox" class="text-[15px] text-muted-foreground" title="Todas las cuentas" aria-label="Todas las cuentas" />
                }
                <ng-icon name="lucideChevronDown" class="text-[16px] text-muted-foreground" />
              </button>
              <ng-template #accountMenu>
                <hlm-dropdown-menu class="w-56">
                  @if (accountHealth().length !== 1) {
                    <button hlmDropdownMenuItem type="button" (click)="selectAccount('all')">
                      <ng-icon name="lucideInbox" />
                      Todas las cuentas
                    </button>
                  }
                  @for (account of accountHealth(); track account.id) {
                    <button hlmDropdownMenuItem type="button" (click)="selectAccount(account.id)">
                      <ng-icon name="lucideMail" [class.text-primary]="filters().accountId === account.id" />
                      <span class="flex-1 truncate">{{ account.email_address }}</span>
                      <ng-icon
                        [name]="connectionStatusIcon(account)"
                        class="text-[15px]"
                        [ngClass]="connectionStatusIconClasses(account)"
                        [title]="connectionStatusLabel(account)"
                        [attr.aria-label]="connectionStatusLabel(account)"
                      />
                    </button>
                  }
                  <hlm-separator class="my-1" />
                  <button hlmDropdownMenuItem type="button" (click)="navigateToAddAccount()">
                    <ng-icon name="lucideCirclePlus" />
                    Añadir cuenta
                  </button>
                </hlm-dropdown-menu>
              </ng-template>
            } @else {
              <button type="button" hlmBtn variant="link" size="sm" class="gap-1.5" (click)="navigateToAddAccount()">
                <ng-icon name="lucideCirclePlus" class="text-[16px]" />
                <span>Añadir cuenta</span>
              </button>
            }
          </div>

          <div class="relative">
            <ng-icon name="lucideSearch" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[18px] text-muted-foreground" />
            <input hlmInput type="search" placeholder="Buscar..." class="pl-9 pr-8" [ngModel]="filters().search" (ngModelChange)="setSearchFilter($event)" />
            <span class="absolute right-3 top-1/2 -translate-y-1/2 rounded border border-border bg-background px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">⌘K</span>
          </div>

          <div class="flex items-center gap-2">
            <button type="button" hlmBtn class="flex-1 gap-1.5" (click)="setFolder('inbox')"><ng-icon name="lucideInbox" class="text-[16px]" /> Inbox</button>
            @if (canSend()) {
              <button type="button" hlmBtn variant="outline" size="icon-sm" (click)="openCompose()" title="Redactar">
                <ng-icon name="lucidePenLine" class="text-[16px]" />
              </button>
            }
            <button type="button" hlmBtn variant="outline" size="icon-sm" [class.bg-muted]="filters().onlyInvoices" (click)="setOnlyInvoicesFilter(!filters().onlyInvoices)" title="Solo facturas">
              <ng-icon name="lucideReceipt" class="text-[16px]" />
            </button>
            <button type="button" hlmBtn variant="outline" size="icon-sm" [disabled]="isSyncing() || syncRetrySecondsLeft() > 0" (click)="triggerSync()" title="Sincronizar correos">
              <ng-icon name="lucideRefreshCw" class="text-[16px]" [class.animate-spin]="isSyncing()" />
            </button>
          </div>

          <div class="flex flex-wrap gap-1">
            @for (folder of folders; track folder.id) {
              <button type="button" hlmBtn size="sm" [variant]="filters().folder === folder.id ? 'default' : 'ghost'" (click)="setFolder(folder.id)">
                {{ folder.label }}
              </button>
            }
          </div>
        </div>

        <div class="relative flex-1 overflow-y-auto">
          @if (error()) {
            <div class="sticky top-0 z-10 bg-background/80 p-4 backdrop-blur-sm">
              <hlm-alert variant="destructive">
                <ng-icon hlm name="lucideCircleAlert" />
                <h4 hlmAlertTitle>Error de conexión</h4>
                <p hlmAlertDescription>{{ error() }}</p>
                <button hlmAlertAction hlmBtn variant="ghost" size="icon" type="button" (click)="clearError()">
                  <ng-icon name="lucideX" />
                </button>
              </hlm-alert>
            </div>
          }

          @if (syncActionError(); as syncError) {
            <div class="sticky top-0 z-10 bg-background/80 p-4 backdrop-blur-sm">
              <hlm-alert [variant]="syncRetrySecondsLeft() > 0 ? 'default' : 'destructive'">
                <ng-icon hlm [name]="syncRetrySecondsLeft() > 0 ? 'lucideTriangleAlert' : 'lucideCircleAlert'" />
                <h4 hlmAlertTitle>{{ syncError.title }}</h4>
                <p hlmAlertDescription>{{ syncError.message }}</p>
                <div class="col-span-2 mt-3 flex flex-wrap items-center gap-2">
                  @if (syncError.requiresReauth) {
                    <button type="button" hlmBtn size="sm" (click)="reauthenticateProvider(syncError.provider)">Reconectar {{ providerLabelFromCode(syncError.provider) }}</button>
                  }
                  @if (!syncError.requiresReauth) {
                    <button type="button" hlmBtn variant="outline" size="sm" [disabled]="syncRetrySecondsLeft() > 0" (click)="triggerSync()">
                      {{ syncRetrySecondsLeft() > 0 ? 'Reintentar en ' + formatRetryCountdown(syncRetrySecondsLeft()) : 'Reintentar ahora' }}
                    </button>
                  }
                  @if (syncError.helpUrl) {
                    <a class="text-xs font-medium text-primary underline underline-offset-2 hover:text-primary/80" [href]="syncError.helpUrl" target="_blank" rel="noopener noreferrer">Ver ayuda</a>
                  }
                </div>
                <button hlmAlertAction hlmBtn variant="ghost" size="icon" type="button" (click)="clearSyncActionError()">
                  <ng-icon name="lucideX" />
                </button>
              </hlm-alert>
            </div>
          }

          @if (loading()) {
            <div class="flex items-center justify-center gap-2 p-8 text-center text-sm text-muted-foreground">
              <hlm-spinner class="size-5" />
              Cargando mensajes...
            </div>
          } @else if (accountHealth().length > 0 && filteredMessages().length === 0) {
            <div class="p-8 text-center">
              <p class="text-sm font-medium text-foreground">No hay mensajes.</p>
            </div>
          } @else if (accountHealth().length === 0) {
            <div class="flex h-40 flex-col items-center justify-center p-8 text-center">
              <p class="text-sm font-medium text-foreground">No hay cuentas conectadas.</p>
            </div>
          } @else if (filteredMessages().length > 0) {
            <div class="flex items-center justify-between px-4 py-3 text-xs font-medium text-muted-foreground">
              <span>{{ currentFolderLabel() }} [{{ filteredMessages().length }}]</span>
            </div>

            <ul class="divide-y divide-border" role="listbox" aria-label="Mensajes del inbox">
              @for (message of filteredMessages(); track message.id) {
                <li
                  role="option"
                  tabindex="0"
                  [attr.aria-selected]="selectedMessage()?.id === message.id"
                  (click)="selectMessage(message)"
                  (keydown)="onMessageKeydown($event, message)"
                  class="relative flex cursor-pointer items-start gap-3 p-4 transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  [class.bg-primary/10]="selectedMessage()?.id === message.id"
                >
                  @if (selectedMessage()?.id === message.id) {
                    <div class="absolute bottom-0 left-0 top-0 w-0.5 bg-primary"></div>
                  }

                  <div class="relative flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted text-sm font-medium text-foreground">
                    @if (message.provider === 'gmail') {
                      <img src="https://www.gstatic.com/images/branding/product/1x/gmail_32dp.png" alt="Gmail" class="size-5 opacity-80" />
                    } @else {
                      {{ message.sender.charAt(0) | uppercase }}
                    }
                    @if (!message.is_read) {
                      <div class="absolute bottom-0 right-0 size-2.5 rounded-full border-2 border-background bg-primary"></div>
                    }
                  </div>

                  <div class="min-w-0 flex-1">
                    <div class="mb-0.5 flex items-baseline justify-between">
                      <span class="truncate pr-2 text-sm text-foreground" [class.font-semibold]="!message.is_read">{{ message.sender }}</span>
                      <span class="shrink-0 text-[11px] text-muted-foreground">{{ message.received_at | date: 'MMM d' }}</span>
                    </div>
                    <p class="truncate text-sm text-muted-foreground" [class.font-medium]="!message.is_read" [class.text-foreground]="!message.is_read">
                      {{ message.subject || '(Sin asunto)' }}
                    </p>

                    <div class="mt-1.5 flex h-4 items-center justify-between">
                      <div class="flex items-center gap-1.5">
                        @if (message.has_xml) {
                          <span hlmBadge variant="outline" class="text-[10px]">XML</span>
                        }
                        @if (message.has_pdf) {
                          <span hlmBadge variant="outline" class="text-[10px]">PDF</span>
                        }
                      </div>
                      <div class="flex items-center gap-1 text-muted-foreground">
                        @if (message.is_starred) {
                          <ng-icon name="lucideStar" class="text-[14px] text-amber-500" />
                        }
                        @if (message.processing_status === 'error') {
                          <ng-icon name="lucideTriangleAlert" class="text-[14px] text-amber-500" />
                        }
                      </div>
                    </div>
                  </div>
                </li>
              }
            </ul>
          }
        </div>
      </aside>

      <main class="relative flex flex-1 flex-col overflow-hidden bg-background">
        @if (selectedMessage(); as message) {
          <div class="flex-1 overflow-y-auto">
            <div class="sticky top-0 z-10 flex h-14 items-center gap-4 border-b border-border bg-background px-6">
              <button type="button" hlmBtn variant="ghost" size="icon-sm" (click)="selectedMessage.set(null)">
                <ng-icon name="lucideArrowLeft" />
              </button>
              <hlm-separator orientation="vertical" class="h-4" />
              <button type="button" hlmBtn variant="ghost" size="icon-sm" title="Archivar" (click)="archiveMessage(message)">
                <ng-icon name="lucideArchive" class="text-[20px]" />
              </button>
              <button type="button" hlmBtn variant="ghost" size="icon-sm" title="Eliminar" (click)="trashMessage(message)">
                <ng-icon name="lucideTrash2" class="text-[20px]" />
              </button>
              <button type="button" hlmBtn variant="ghost" size="icon-sm" [title]="message.is_read ? 'Marcar no leído' : 'Marcar leído'" (click)="toggleRead(message)">
                <ng-icon [name]="message.is_read ? 'lucideMail' : 'lucideMailOpen'" class="text-[20px]" />
              </button>
              @if (canSend()) {
                <button type="button" hlmBtn variant="ghost" size="icon-sm" title="Responder" (click)="openReply(message)">
                  <ng-icon name="lucidePenLine" class="text-[20px]" />
                </button>
              }
            </div>

            <div class="min-h-full w-full px-8 py-10">
              <h1 class="mb-8 text-2xl font-semibold">{{ message.subject || '(Sin asunto)' }}</h1>

              <div class="mb-8 flex items-start justify-between">
                <div class="flex items-center gap-3">
                  <div class="flex size-10 shrink-0 items-center justify-center rounded-full bg-muted font-medium text-foreground">
                    {{ message.sender.charAt(0) | uppercase }}
                  </div>
                  <div>
                    <div class="font-medium">{{ message.sender }}</div>
                    <div class="flex items-center gap-1 text-xs text-muted-foreground">
                      Para: <span class="font-medium text-foreground">{{ message.account_email }}</span>
                    </div>
                  </div>
                </div>
                <div class="flex items-center gap-2 text-sm text-muted-foreground">
                  {{ message.received_at | date: 'medium' }}
                  <button type="button" hlmBtn variant="ghost" size="icon-sm" [title]="message.is_starred ? 'Quitar estrella' : 'Destacar'" (click)="toggleStar(message)">
                    <ng-icon name="lucideStar" class="text-[18px]" [class.text-amber-500]="message.is_starred" />
                  </button>
                </div>
              </div>

              <div class="prose prose-sm dark:prose-invert max-w-none whitespace-pre-wrap text-foreground">
                @if (isDetailLoading()) {
                  Cargando contenido del correo...
                } @else if (selectedMessageHtml()) {
                  <app-secure-email-body [html]="selectedMessageHtml() || ''" />
                } @else {
                  {{ selectedMessageBody() || 'Este mensaje no contiene texto plano o aún no se ha extraído.' }}
                }
              </div>

              @if (detailError()) {
                <p class="mt-3 text-sm text-destructive">{{ detailError() }}</p>
              }

              @if (selectedAttachments().length > 0) {
                <hlm-card class="mt-10 p-4">
                  <h3 class="mb-3 text-sm font-semibold">Adjuntos</h3>
                  <div class="flex flex-wrap gap-3">
                    @for (attachment of selectedAttachments(); track attachment.id) {
                      <button
                        type="button"
                        class="flex w-64 items-center gap-3 rounded-lg border bg-card p-3 text-left shadow-sm transition-colors hover:border-border/80"
                        (click)="downloadAttachment(message.id, attachment.id, attachment.filename)"
                      >
                        <div class="flex size-10 shrink-0 items-center justify-center rounded bg-muted text-muted-foreground">
                          <ng-icon name="lucidePaperclip" class="text-[20px]" />
                        </div>
                        <div class="min-w-0">
                          <p class="truncate text-sm font-medium">{{ attachment.filename }}</p>
                          <p class="text-xs text-muted-foreground">{{ attachment.mime_type || 'Adjunto' }}</p>
                        </div>
                      </button>
                    }
                  </div>
                </hlm-card>
              }
            </div>
          </div>
        } @else {
          <div class="absolute inset-0 flex items-center justify-center p-8">
            <div class="mx-auto max-w-sm text-center">
              @if (accountHealth().length > 0) {
                <div class="relative mx-auto mb-6 flex size-32 items-center justify-center rounded-full bg-muted shadow-[inset_0_2px_4px_rgba(0,0,0,0.02)]">
                  <div class="flex h-20 w-16 -rotate-6 flex-col gap-2 rounded-lg border bg-card p-2.5 shadow-sm transition-all duration-300 hover:rotate-0">
                    <div class="flex items-center gap-1 border-b border-border pb-2">
                      <ng-icon name="lucideMail" class="text-[10px] text-muted-foreground" />
                      <div class="h-1.5 w-6 rounded-full bg-muted"></div>
                    </div>
                    <div class="h-1.5 w-full rounded-full bg-muted"></div>
                    <div class="h-1.5 w-2/3 rounded-full bg-muted"></div>
                  </div>
                </div>
                <h3 class="text-lg font-medium">No hay mensajes</h3>
                <p class="mt-1.5 text-sm text-muted-foreground">Elige un correo para ver los detalles</p>
              } @else {
                <div class="relative mx-auto mb-6 flex size-32 items-center justify-center rounded-full bg-primary/10">
                  <ng-icon name="lucideMailOpen" class="text-[48px] text-primary" />
                </div>
                <h3 class="text-lg font-medium">Empieza a recibir tus correos</h3>
                <p class="mb-6 mt-1.5 text-sm text-muted-foreground">Conecta tu cuenta de correo para centralizar tus facturas y documentos.</p>
                <button type="button" hlmBtn class="gap-2" (click)="navigateToAddAccount()">
                  <ng-icon name="lucideCirclePlus" class="text-[18px]" />
                  Conectar cuenta
                </button>
              }
            </div>
          </div>
        }
      </main>

      @if (canSend() && composeOpen()) {
        <div class="absolute bottom-4 right-4 z-20 w-[420px] rounded-xl border border-border bg-card shadow-lg">
          <div class="flex items-center justify-between border-b border-border px-4 py-3">
            <h3 class="text-sm font-semibold">{{ replyToMessage() ? 'Responder' : 'Redactar' }}</h3>
            <button type="button" hlmBtn variant="ghost" size="icon-sm" (click)="closeCompose()">
              <ng-icon name="lucideX" />
            </button>
          </div>
          <form class="space-y-3 p-4" (submit)="submitCompose($event)">
            @if (accountHealth().length > 1) {
              <select hlmInput class="w-full" [ngModel]="composeAccountId()" (ngModelChange)="composeAccountId.set($event)" name="composeAccount">
                @for (account of accountHealth(); track account.id) {
                  <option [value]="account.id">{{ account.email_address }}</option>
                }
              </select>
            }
            <input hlmInput type="text" placeholder="Para" class="w-full" [ngModel]="composeTo()" (ngModelChange)="composeTo.set($event)" name="composeTo" required />
            <input hlmInput type="text" placeholder="Asunto" class="w-full" [ngModel]="composeSubject()" (ngModelChange)="composeSubject.set($event)" name="composeSubject" />
            <textarea hlmInput placeholder="Mensaje" class="min-h-32 w-full" [ngModel]="composeBody()" (ngModelChange)="composeBody.set($event)" name="composeBody"></textarea>
            <div class="flex justify-end">
              <button type="submit" hlmBtn class="gap-2" [disabled]="sending()">
                <ng-icon name="lucideSend" class="text-[16px]" />
                Enviar
              </button>
            </div>
          </form>
        </div>
      }
    </div>
  `,
})
export class MasterInboxPage implements OnInit, OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly store = inject(UnifiedInboxStore);
  private readonly entitlements = inject(EntitlementsStore);
  readonly canSend = this.entitlements.hasMailSend;

  readonly loading = this.store.loading;
  readonly error = this.store.error;
  readonly detailError = this.store.detailError;
  readonly loadingMessageId = this.store.loadingMessageId;
  readonly tenantId = signal('');
  readonly accountHealth = this.store.accountHealth;
  readonly filteredMessages = this.store.filteredMessages;
  readonly filters = this.store.filters;
  readonly isSyncing = this.store.isSyncing;
  readonly syncActionError = this.store.syncActionError;
  readonly syncRetrySecondsLeft = this.store.syncRetrySecondsLeft;
  readonly composeOpen = this.store.composeOpen;
  readonly sending = this.store.sending;

  readonly folders: { id: MailFolder; label: string }[] = [
    { id: 'inbox', label: 'Inbox' },
    { id: 'starred', label: 'Destacados' },
    { id: 'sent', label: 'Enviados' },
    { id: 'drafts', label: 'Borradores' },
    { id: 'archive', label: 'Archivo' },
    { id: 'trash', label: 'Papelera' },
  ];

  readonly selectedMessage = signal<UnifiedInboxMessage | null>(null);
  readonly replyToMessage = signal<UnifiedInboxMessage | null>(null);
  readonly composeTo = signal('');
  readonly composeSubject = signal('');
  readonly composeBody = signal('');
  readonly composeAccountId = signal('');
  readonly selectedMessageBody = computed(() => {
    const selected = this.selectedMessage();
    if (!selected) {
      return '';
    }

    const detail = this.store.getMessageDetail(selected.id);
    if (!detail) {
      return selected.snippet || '';
    }

    return detail.body_text || '';
  });
  readonly selectedMessageHtml = computed(() => {
    const selected = this.selectedMessage();
    if (!selected) {
      return '';
    }

    const detail = this.store.getMessageDetail(selected.id);
    return detail?.body_html || '';
  });
  readonly isDetailLoading = computed(() => {
    const selected = this.selectedMessage();
    return !!selected && this.loadingMessageId() === selected.id;
  });
  readonly selectedAttachments = computed(() => {
    const selected = this.selectedMessage();
    if (!selected) {
      return [];
    }

    return this.store.getMessageDetail(selected.id)?.attachments ?? [];
  });

  readonly currentAccountLabel = computed(() => {
    const accountId = this.filters().accountId;
    if (accountId === 'all') return 'Todas las cuentas';
    const account = this.accountHealth().find((a) => a.id === accountId);
    return account ? account.email_address : 'Todas las cuentas';
  });

  readonly selectedAccount = computed(() => {
    const accountId = this.filters().accountId;
    if (accountId === 'all') return null;
    return this.accountHealth().find((account) => account.id === accountId) ?? null;
  });

  ngOnInit(): void {
    this.tenantId.set(this.route.snapshot.paramMap.get('tenantId') || '');
    this.store.init();
  }

  ngOnDestroy(): void {
    this.store.destroy();
  }

  providerLabelFromCode(providerCode: string | undefined): string {
    const normalized = (providerCode || '').trim().toLowerCase() as MailProvider;
    if (!normalized) {
      return 'cuenta';
    }

    if (!['gmail', 'microsoft', 'outlook', 'hotmail', 'yahoo'].includes(normalized)) {
      return 'cuenta';
    }

    return providerLabel(normalized);
  }

  currentFolderLabel(): string {
    return this.folders.find((folder) => folder.id === this.filters().folder)?.label ?? 'Inbox';
  }

  setFolder(folder: MailFolder): void {
    this.store.setFolder(folder);
    this.selectedMessage.set(null);
  }

  setSearchFilter(search: string): void {
    this.store.patchFilters({ search });
  }

  setOnlyInvoicesFilter(onlyInvoices: boolean): void {
    this.store.patchFilters({ onlyInvoices });
  }

  triggerSync(): void {
    this.store.triggerSync();
  }

  selectAccount(accountId: string): void {
    this.store.patchFilters({ accountId });
  }

  navigateToAddAccount(): void {
    void this.router.navigate(['/', this.tenantId(), 'connections']);
  }

  clearError(): void {
    this.store.clearError();
  }

  clearSyncActionError(): void {
    this.store.clearSyncActionError();
  }

  reauthenticateProvider(providerCode: string | undefined): void {
    this.store.reauthenticateProvider(providerCode, () => this.navigateToAddAccount());
  }

  formatRetryCountdown(totalSeconds: number): string {
    if (totalSeconds <= 0) {
      return '00:00';
    }

    const minutes = Math.floor(totalSeconds / 60)
      .toString()
      .padStart(2, '0');
    const seconds = (totalSeconds % 60).toString().padStart(2, '0');
    return `${minutes}:${seconds}`;
  }

  selectMessage(message: UnifiedInboxMessage): void {
    this.selectedMessage.set(message);
    this.store.loadMessageDetail(message.id);
    if (!message.is_read) {
      this.store.modifyMessage(message.id, 'read');
      this.selectedMessage.update((current) => (current ? { ...current, is_read: true } : current));
    }
  }

  toggleStar(message: UnifiedInboxMessage): void {
    this.store.modifyMessage(message.id, message.is_starred ? 'unstar' : 'star');
    this.selectedMessage.update((current) => (current?.id === message.id ? { ...current, is_starred: !message.is_starred } : current));
  }

  toggleRead(message: UnifiedInboxMessage): void {
    this.store.modifyMessage(message.id, message.is_read ? 'unread' : 'read');
    this.selectedMessage.update((current) => (current?.id === message.id ? { ...current, is_read: !message.is_read } : current));
  }

  archiveMessage(message: UnifiedInboxMessage): void {
    this.store.modifyMessage(message.id, 'archive');
    this.selectedMessage.set(null);
  }

  trashMessage(message: UnifiedInboxMessage): void {
    this.store.modifyMessage(message.id, 'trash');
    this.selectedMessage.set(null);
  }

  downloadAttachment(messageId: string, attachmentId: string, filename: string): void {
    this.store.downloadAttachment(messageId, attachmentId, filename);
  }

  openCompose(): void {
    this.replyToMessage.set(null);
    this.composeTo.set('');
    this.composeSubject.set('');
    this.composeBody.set('');
    this.composeAccountId.set(this.defaultComposeAccountId());
    this.store.composeOpen.set(true);
  }

  openReply(message: UnifiedInboxMessage): void {
    const subject = message.subject || '';
    this.replyToMessage.set(message);
    this.composeTo.set(message.sender);
    this.composeSubject.set(subject.toLowerCase().startsWith('re:') ? subject : `Re: ${subject}`.trim());
    this.composeBody.set('');
    this.composeAccountId.set(message.account_id);
    this.store.composeOpen.set(true);
  }

  closeCompose(): void {
    this.store.composeOpen.set(false);
    this.replyToMessage.set(null);
  }

  submitCompose(event: Event): void {
    event.preventDefault();
    const accountId = this.composeAccountId() || this.defaultComposeAccountId();
    const to = this.composeTo()
      .split(',')
      .map((value) => value.trim())
      .filter(Boolean);
    if (!accountId || to.length === 0) {
      return;
    }

    const reply = this.replyToMessage();
    this.store.sendMessage({
      account_id: accountId,
      to,
      subject: this.composeSubject(),
      body_text: this.composeBody(),
      thread_id: reply?.thread_id,
      in_reply_to: reply?.id,
    });
  }

  private defaultComposeAccountId(): string {
    const selected = this.filters().accountId;
    if (selected && selected !== 'all') {
      return selected;
    }

    return this.accountHealth()[0]?.id ?? '';
  }

  onMessageKeydown(event: KeyboardEvent, message: UnifiedInboxMessage): void {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      this.selectMessage(message);
    }
  }

  connectionStatusIcon(account: AccountHealthSummary): string {
    return connectionStatusIcon(resolveConnectionStatus(account));
  }

  connectionStatusIconClasses(account: AccountHealthSummary): string {
    return connectionStatusIconClasses(resolveConnectionStatus(account));
  }

  connectionStatusLabel(account: AccountHealthSummary): string {
    return connectionStatusDetailLabel(resolveConnectionStatus(account));
  }
}
