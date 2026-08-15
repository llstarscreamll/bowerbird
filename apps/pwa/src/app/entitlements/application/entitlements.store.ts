import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, catchError, of, tap } from 'rxjs';
import { FeatureKeys, TenantEntitlements } from '../domain/entitlements.model';
import { EntitlementsHttpRepository } from '../infrastructure/entitlements.http.repository';

@Injectable({ providedIn: 'root' })
export class EntitlementsStore {
  private readonly repo = inject(EntitlementsHttpRepository);

  readonly features = signal<string[]>([]);
  readonly loadedTenantId = signal('');

  readonly hasMailInbox = computed(() => this.features().includes(FeatureKeys.MailInbox));
  readonly hasMailSend = computed(() => this.features().includes(FeatureKeys.MailSend));
  readonly hasInvoicing = computed(() => this.features().includes(FeatureKeys.InvoicingWorkspace) || this.features().includes(FeatureKeys.InvoicingCaptureFromEmail));
  readonly showConnections = computed(() => this.hasMailInbox() || this.features().includes(FeatureKeys.InvoicingCaptureFromEmail));

  has(featureKey: string): boolean {
    return this.features().includes(featureKey);
  }

  load(tenantId: string): Observable<TenantEntitlements | null> {
    return this.repo.getEntitlements().pipe(
      tap((response) => {
        this.features.set(response.features ?? []);
        this.loadedTenantId.set(tenantId);
      }),
      catchError(() => {
        this.features.set([]);
        this.loadedTenantId.set(tenantId);
        return of(null);
      }),
    );
  }
}
