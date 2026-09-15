import { Injectable, computed, inject, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable, catchError, map, of, tap } from 'rxjs';
import { ToastService } from '../../core/services/toast.service';
import { CatalogHttpService } from '../infrastructure/catalog.http.service';
import { CatalogAlias, CatalogItem, CreateCatalogAliasInput, CreateCatalogItemInput, DuplicateCluster, MergeItemsInput, UpdateCatalogItemInput } from '../domain/catalog.model';
import { isEnrichedHttpError } from '../../core/http/jsonapi-error';

@Injectable({ providedIn: 'root' })
export class CatalogStore {
  private readonly http = inject(CatalogHttpService);
  private readonly toast = inject(ToastService);

  readonly items = signal<CatalogItem[]>([]);
  readonly selectedItem = signal<CatalogItem | null>(null);
  readonly duplicateClusters = signal<DuplicateCluster[]>([]);
  readonly loading = signal(false);
  readonly loadingMore = signal(false);
  readonly submitting = signal(false);
  readonly errorMessage = signal<string | null>(null);
  readonly conflictOwnerItemId = signal<string | null>(null);
  readonly hasMore = signal(false);
  readonly search = signal('');
  private cursor = signal<string | undefined>(undefined);

  loadItems(status?: string): void {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.cursor.set(undefined);
    this.http.listItems({ status, search: this.search() || undefined, pageSize: 50 }).subscribe({
      next: (page) => {
        this.items.set(page.items);
        this.hasMore.set(page.hasMore);
        this.cursor.set(page.cursor);
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => this.handleError(err, 'No se pudieron cargar los ítems.'),
    });
  }

  loadMore(status?: string): void {
    if (!this.hasMore() || this.loadingMore()) return;
    this.loadingMore.set(true);
    this.http.listItems({ status, search: this.search() || undefined, pageSize: 50, after: this.cursor() }).subscribe({
      next: (page) => {
        this.items.update((current) => [...current, ...page.items]);
        this.hasMore.set(page.hasMore);
        this.cursor.set(page.cursor);
        this.loadingMore.set(false);
      },
      error: (err: HttpErrorResponse) => {
        this.loadingMore.set(false);
        this.handleError(err, 'No se pudieron cargar más ítems.');
      },
    });
  }

  loadItem(id: string): void {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.conflictOwnerItemId.set(null);
    this.selectedItem.set(null);
    this.http.getItem(id).subscribe({
      next: (item) => {
        this.selectedItem.set(item);
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => this.handleError(err, 'No se pudo cargar el ítem.'),
    });
  }

  createItem(input: CreateCatalogItemInput): Observable<CatalogItem | null> {
    this.submitting.set(true);
    this.errorMessage.set(null);
    return this.http.createItem(input).pipe(
      tap((item) => {
        this.submitting.set(false);
        this.selectedItem.set(item);
        this.toast.showSuccess('Ítem creado.');
      }),
      catchError((err: HttpErrorResponse) => {
        this.submitting.set(false);
        this.handleError(err, 'No se pudo crear el ítem.');
        return of(null);
      }),
    );
  }

  updateItem(id: string, input: UpdateCatalogItemInput): Observable<CatalogItem | null> {
    this.submitting.set(true);
    this.errorMessage.set(null);
    this.conflictOwnerItemId.set(null);
    return this.http.updateItem(id, input).pipe(
      tap((item) => {
        this.submitting.set(false);
        this.selectedItem.set(item);
        this.toast.showSuccess('Ítem actualizado.');
      }),
      catchError((err: HttpErrorResponse) => {
        this.submitting.set(false);
        this.handleError(err, 'No se pudo actualizar el ítem.');
        return of(null);
      }),
    );
  }

  addAlias(itemId: string, input: CreateCatalogAliasInput): Observable<CatalogAlias | null> {
    this.submitting.set(true);
    this.errorMessage.set(null);
    this.conflictOwnerItemId.set(null);
    return this.http.addAlias(itemId, input).pipe(
      tap(() => {
        this.submitting.set(false);
        this.toast.showSuccess('Alias añadido.');
        this.loadItem(itemId);
      }),
      catchError((err: unknown) => {
        this.submitting.set(false);
        this.handleError(err, 'No se pudo añadir el alias.');
        return of(null);
      }),
    );
  }

  removeAlias(itemId: string, aliasId: string): Observable<boolean> {
    this.submitting.set(true);
    this.errorMessage.set(null);
    return this.http.removeAlias(itemId, aliasId).pipe(
      tap(() => {
        this.submitting.set(false);
        this.toast.showSuccess('Alias eliminado.');
        this.loadItem(itemId);
      }),
      map(() => true),
      catchError((err: unknown) => {
        this.submitting.set(false);
        this.handleError(err, 'No se pudo eliminar el alias.');
        return of(false);
      }),
    );
  }

  loadDuplicateClusters(): void {
    this.http.listDuplicateClusters().subscribe({
      next: (clusters) => this.duplicateClusters.set(clusters),
      error: () => this.duplicateClusters.set([]),
    });
  }

  mergeItems(input: MergeItemsInput): Observable<CatalogItem | null> {
    this.submitting.set(true);
    this.errorMessage.set(null);
    return this.http.mergeItems(input).pipe(
      tap((item) => {
        this.submitting.set(false);
        this.selectedItem.set(item);
        this.toast.showSuccess('Ítems fusionados.');
        this.loadDuplicateClusters();
      }),
      catchError((err: HttpErrorResponse) => {
        this.submitting.set(false);
        this.handleError(err, 'No se pudieron fusionar los ítems.');
        return of(null);
      }),
    );
  }

  markNotDuplicates(itemIds: string[]): Observable<boolean> {
    this.submitting.set(true);
    this.errorMessage.set(null);
    return this.http.markNotDuplicates(itemIds).pipe(
      tap(() => {
        this.submitting.set(false);
        this.toast.showSuccess('Marcados como productos distintos.');
        this.loadDuplicateClusters();
      }),
      map(() => true),
      catchError((err: unknown) => {
        this.submitting.set(false);
        this.handleError(err, 'No se pudo guardar la decisión.');
        return of(false);
      }),
    );
  }

  readonly hasItems = computed(() => this.items().length > 0);

  private handleError(err: unknown, fallback: string): void {
    this.loading.set(false);
    this.submitting.set(false);
    const enriched = isEnrichedHttpError(err) ? err : null;
    const http = enriched?.original ?? (err instanceof HttpErrorResponse ? err : null);
    const first = enriched?.jsonApiErrors?.[0];
    const owner = typeof first?.meta?.['item_id'] === 'string' ? String(first.meta['item_id']) : null;
    this.conflictOwnerItemId.set(owner);
    if (http && http.status >= 400 && http.status < 500) {
      this.errorMessage.set(first?.detail || http.error?.errors?.[0]?.detail || fallback);
    } else {
      this.toast.showError(fallback);
    }
  }
}
