import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, catchError, of, tap } from 'rxjs';
import { MyPermissions, PermissionCodes } from '../domain/rbac.model';
import { RbacHttpRepository } from '../infrastructure/rbac.http.repository';

@Injectable({ providedIn: 'root' })
export class PermissionsStore {
  private readonly repo = inject(RbacHttpRepository);

  readonly permissions = signal<string[]>([]);
  readonly loadedTenantId = signal('');

  readonly canReadSecrets = computed(() => this.has(PermissionCodes.SecretsRead));
  readonly canWriteSecrets = computed(() => this.has(PermissionCodes.SecretsWrite));
  readonly canDeleteSecrets = computed(() => this.has(PermissionCodes.SecretsDelete));

  has(code: string): boolean {
    return this.permissions().includes(code);
  }

  load(tenantId: string): Observable<MyPermissions | null> {
    return this.repo.getMyPermissions().pipe(
      tap((response) => {
        this.permissions.set(response.permissions ?? []);
        this.loadedTenantId.set(tenantId);
      }),
      catchError(() => {
        this.permissions.set([]);
        this.loadedTenantId.set(tenantId);
        return of(null);
      }),
    );
  }
}
