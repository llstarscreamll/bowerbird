import { Injectable, computed, inject, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable, catchError, map, of, tap } from 'rxjs';
import { ToastService } from '../../core/services/toast.service';
import { CatalogHttpService } from '../infrastructure/catalog.http.service';
import { CatalogItem, CreateCatalogItemInput, UpdateCatalogItemInput } from '../domain/catalog.model';

@Injectable({ providedIn: 'root' })
export class CatalogStore {
  private readonly http = inject(CatalogHttpService);
  private readonly toast = inject(ToastService);

  readonly items = signal<CatalogItem[]>([]);
  readonly selectedItem = signal<CatalogItem | null>(null);
  readonly loading = signal(false);
  readonly loadingMore = signal(false);
  readonly submitting = signal(false);
  readonly errorMessage = signal<string | null>(null);
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
    this.selectedItem.set(null);
    this.http.getItem(id).subscribe({
      next: (item) => {
        this.selectedItem.set(item);
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => this.handleError(err, 'No se pudo cargar el ítem.'),
    });
  }

  searchItems(query: string): Observable<CatalogItem[]> {
    return this.http.listItems({ search: query, pageSize: 20 }).pipe(
      map((page) => page.items),
      catchError((err: HttpErrorResponse) => {
        this.handleError(err, 'No se pudo buscar en el catálogo.');
        return of([]);
      }),
    );
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

  readonly hasItems = computed(() => this.items().length > 0);

  private handleError(err: HttpErrorResponse, fallback: string): void {
    this.loading.set(false);
    this.submitting.set(false);
    if (err.status >= 400 && err.status < 500) {
      this.errorMessage.set(err.error?.errors?.[0]?.detail || fallback);
    } else {
      this.toast.showError(fallback);
    }
  }
}
