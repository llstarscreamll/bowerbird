import { CommonModule, CurrencyPipe, DatePipe } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { InvoiceDetailsStore } from '../../../application/invoice-details.store';
import { CatalogLinkerComponent } from '../../components/catalog-linker/catalog-linker.component';
import { InvoiceLine } from '../../../domain/invoice.model';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSeparatorImports } from '@spartan-ng/helm/separator';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';

@Component({
  selector: 'app-invoices-detail',
  standalone: true,
  imports: [CommonModule, CurrencyPipe, DatePipe, RouterLink, NgIcon, HlmCardImports, HlmButtonImports, HlmSpinnerImports, HlmSeparatorImports, HlmBadgeImports, CatalogLinkerComponent],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto bg-muted/20 p-4 sm:p-6 lg:p-8' },
  template: `
    <div class="mx-auto w-full max-w-7xl space-y-8">
      @if (isLoading()) {
        <div class="flex items-center justify-center py-20">
          <hlm-spinner class="size-8 text-primary" />
        </div>
      }

      @if (invoice(); as inv) {
        <a routerLink="../" class="inline-flex items-center gap-1.5 text-sm font-medium text-muted-foreground hover:text-foreground">
          <ng-icon name="lucideArrowLeft" class="size-4" />
          Volver a facturas
        </a>

        <header class="flex flex-col gap-4 border-b border-border pb-6 lg:flex-row lg:items-end lg:justify-between">
          <div class="min-w-0 space-y-2">
            <div class="flex flex-wrap items-center gap-2">
              <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">Factura {{ inv.invoice_number || 'Sin número' }}</h1>
              @if (inv.linking_status) {
                <span hlmBadge [variant]="linkingBadgeVariant(inv.linking_status)">{{ linkingStatusLabel(inv.linking_status) }}</span>
              }
              @if (inv.extraction_source) {
                <span hlmBadge variant="outline">{{ inv.extraction_source === 'xml' ? 'XML DIAN' : inv.extraction_source }}</span>
              }
            </div>
            <p class="text-sm text-muted-foreground">
              Emitida el
              <span class="font-medium text-foreground">{{ inv.issue_date ? (inv.issue_date | date: 'longDate') : 'fecha desconocida' }}</span>
              · {{ inv.currency_code || 'COP' }}
            </p>
          </div>
          <button hlmBtn variant="outline" class="shrink-0 self-start lg:self-auto">
            <ng-icon name="lucideDownload" />
            Descargar
          </button>
        </header>

        <section class="grid grid-cols-1 gap-4 lg:grid-cols-12">
          <hlm-card class="space-y-4 p-5 lg:col-span-4">
            <h2 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Emisor</h2>
            <div>
              <p class="text-base font-semibold leading-snug">{{ inv.issuer_name || 'Desconocido' }}</p>
              <p class="mt-1 text-sm text-muted-foreground">NIT {{ inv.issuer_tax_id || '—' }}</p>
            </div>
            @if (inv.issuer_party_id) {
              <a class="inline-flex items-center gap-1.5 text-sm text-primary hover:underline" [routerLink]="['/', tenantId(), 'parties']">
                <ng-icon name="lucideBuilding2" class="size-3.5" />
                Contacto en catálogo
              </a>
            }
          </hlm-card>

          <hlm-card class="space-y-4 p-5 lg:col-span-4">
            <h2 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Receptor</h2>
            <div>
              <p class="text-base font-semibold leading-snug">{{ inv.receiver_name || 'Desconocido' }}</p>
              <p class="mt-1 text-sm text-muted-foreground">NIT {{ inv.receiver_tax_id || '—' }}</p>
            </div>
            <div class="grid grid-cols-2 gap-3 border-t border-border pt-4">
              <div>
                <p class="text-xs text-muted-foreground">Vencimiento</p>
                <p class="mt-0.5 text-sm font-medium">{{ inv.due_date ? (inv.due_date | date: 'mediumDate') : '—' }}</p>
              </div>
              <div>
                <p class="text-xs text-muted-foreground">Pago</p>
                <p class="mt-0.5 text-sm font-medium">{{ inv.payment_code || '—' }}</p>
              </div>
            </div>
          </hlm-card>

          <hlm-card class="flex flex-col justify-between p-5 lg:col-span-4">
            <h2 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Totales</h2>
            <div class="mt-4 flex flex-col gap-2.5">
              <div class="flex justify-between text-sm">
                <span class="text-muted-foreground">Subtotal</span>
                <span class="tabular-nums">{{ inv.subtotal | currency: inv.currency_code : 'symbol' : '1.2-2' }}</span>
              </div>
              <div class="flex justify-between text-sm">
                <span class="text-muted-foreground">Impuestos</span>
                <span class="tabular-nums">{{ inv.tax_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}</span>
              </div>
              @if (inv.allowance_total > 0) {
                <div class="flex justify-between text-sm">
                  <span class="text-muted-foreground">Descuentos</span>
                  <span class="tabular-nums">{{ -inv.allowance_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}</span>
                </div>
              }
              <hlm-separator />
              <div class="flex items-baseline justify-between gap-3 pt-1">
                <span class="text-sm font-semibold">Total</span>
                <span class="text-xl font-semibold tabular-nums text-primary">{{ inv.grand_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}</span>
              </div>
            </div>
          </hlm-card>
        </section>

        <section class="space-y-3">
          <div class="flex flex-wrap items-end justify-between gap-2">
            <div>
              <h2 class="text-lg font-semibold tracking-tight">Líneas</h2>
              <p class="text-sm text-muted-foreground">Montos de la factura y vínculo opcional al catálogo de ítems.</p>
            </div>
            <span class="text-sm text-muted-foreground">{{ inv.lines.length || 0 }} línea(s)</span>
          </div>

          @if (!inv.lines || inv.lines.length === 0) {
            <hlm-card class="px-6 py-12 text-center text-sm text-muted-foreground">No hay líneas en esta factura.</hlm-card>
          } @else {
            <div class="space-y-3">
              @for (line of inv.lines; track line.id || $index; let i = $index) {
                <hlm-card class="overflow-hidden p-0">
                  <div class="grid grid-cols-1 gap-0 lg:grid-cols-12">
                    <div class="space-y-3 border-b border-border p-5 lg:col-span-8 lg:border-b-0 lg:border-r">
                      <div class="flex flex-wrap items-start justify-between gap-2">
                        <div class="min-w-0 flex-1">
                          <p class="text-xs font-medium uppercase tracking-wide text-muted-foreground">Línea {{ line.line_number || i + 1 }}</p>
                          <p class="mt-1 text-base font-medium leading-snug break-words">{{ line.description || 'Sin descripción' }}</p>
                          <p class="mt-1 font-mono text-xs text-muted-foreground">Código proveedor: {{ line.item_code || '—' }}</p>
                        </div>
                        <span hlmBadge [variant]="lineBadgeVariant(line)">{{ lineStatusLabel(line) }}</span>
                      </div>

                      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
                        <div>
                          <p class="text-xs text-muted-foreground">Cantidad</p>
                          <p class="mt-0.5 text-sm font-medium tabular-nums">{{ line.quantity }}</p>
                        </div>
                        <div>
                          <p class="text-xs text-muted-foreground">Precio unit.</p>
                          <p class="mt-0.5 text-sm font-medium tabular-nums">{{ line.unit_price | currency: inv.currency_code : 'symbol' : '1.2-2' }}</p>
                        </div>
                        <div>
                          <p class="text-xs text-muted-foreground">Impuesto</p>
                          <p class="mt-0.5 text-sm font-medium tabular-nums">{{ line.line_tax_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}</p>
                        </div>
                        <div>
                          <p class="text-xs text-muted-foreground">Total línea</p>
                          <p class="mt-0.5 text-sm font-semibold tabular-nums">{{ line.line_total | currency: inv.currency_code : 'symbol' : '1.2-2' }}</p>
                        </div>
                      </div>
                    </div>

                    <div class="flex flex-col justify-between gap-3 bg-muted/30 p-5 lg:col-span-4">
                      <div>
                        <p class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Catálogo</p>
                        @if (line.item_id && line.link_status === 'linked') {
                          <p class="mt-2 text-sm">
                            Ítem
                            <a class="font-mono text-xs text-primary hover:underline" [routerLink]="['/', tenantId(), 'catalog']">{{ shortId(line.item_id) }}</a>
                          </p>
                          @if (line.link_method) {
                            <p class="mt-1 text-xs text-muted-foreground">Método: {{ methodLabel(line.link_method) }}</p>
                          }
                          @if (line.link_locked) {
                            <p class="mt-1 text-xs text-muted-foreground">Bloqueado por el usuario</p>
                          }
                        } @else if (line.link_status === 'rejected') {
                          <p class="mt-2 text-sm text-muted-foreground">Vinculación rechazada.</p>
                        } @else {
                          <p class="mt-2 text-sm text-muted-foreground">Sin ítem de catálogo vinculado.</p>
                        }
                      </div>

                      @if (line.id && needsLinking(line)) {
                        <app-catalog-linker
                          [invoiceId]="inv.id"
                          [lineId]="line.id"
                          [description]="line.description"
                          [itemCode]="line.item_code"
                          [suggestions]="line.suggestions || []"
                          (resolved)="onLineResolved()"
                        />
                      }
                    </div>
                  </div>
                </hlm-card>
              }
            </div>
          }
        </section>

        <section class="space-y-2">
          <h2 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">CUFE</h2>
          <p class="break-all rounded-lg border border-border bg-card px-4 py-3 font-mono text-xs leading-relaxed text-muted-foreground">
            {{ inv.cufe || '—' }}
          </p>
        </section>
      }
    </div>
  `,
})
export class DetailPage implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly store = inject(InvoiceDetailsStore);

  readonly invoice = this.store.invoice;
  readonly isLoading = this.store.isLoading;

  ngOnInit(): void {
    const invoiceId = this.route.snapshot.paramMap.get('invoiceId');
    if (invoiceId) {
      this.store.loadInvoice(invoiceId);
    }
  }

  tenantId(): string {
    return this.route.parent?.snapshot.paramMap.get('tenantId') || this.route.snapshot.paramMap.get('tenantId') || location.pathname.split('/').filter(Boolean)[0] || '';
  }

  shortId(id: string): string {
    if (!id) return '—';
    return id.length > 12 ? `${id.slice(0, 8)}…${id.slice(-4)}` : id;
  }

  needsLinking(line: InvoiceLine): boolean {
    if (line.link_locked) return false;
    return line.link_status === 'unmatched' || line.link_status === 'suggested' || !line.link_status;
  }

  onLineResolved(): void {
    const invoiceId = this.route.snapshot.paramMap.get('invoiceId');
    if (invoiceId) this.store.loadInvoice(invoiceId);
  }

  linkingStatusLabel(status: string): string {
    switch (status) {
      case 'linked':
        return 'Catálogo vinculado';
      case 'failed':
        return 'Vinculación fallida';
      case 'pending':
        return 'Pendiente de catálogo';
      default:
        return status;
    }
  }

  linkingBadgeVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
    if (status === 'failed') return 'destructive';
    if (status === 'linked') return 'default';
    return 'secondary';
  }

  lineStatusLabel(line: InvoiceLine): string {
    switch (line.link_status) {
      case 'linked':
        return 'Vinculada';
      case 'suggested':
        return 'Sugerida';
      case 'rejected':
        return 'Rechazada';
      case 'unmatched':
        return 'Sin vínculo';
      default:
        return line.link_status || 'Sin vínculo';
    }
  }

  lineBadgeVariant(line: InvoiceLine): 'default' | 'secondary' | 'destructive' | 'outline' {
    switch (line.link_status) {
      case 'linked':
        return 'default';
      case 'suggested':
        return 'outline';
      case 'rejected':
        return 'destructive';
      default:
        return 'secondary';
    }
  }

  methodLabel(method: string): string {
    switch (method) {
      case 'memory':
        return 'memoria';
      case 'hard':
        return 'código proveedor';
      case 'soft':
        return 'sugerencia';
      case 'manual':
        return 'manual';
      default:
        return method;
    }
  }
}
