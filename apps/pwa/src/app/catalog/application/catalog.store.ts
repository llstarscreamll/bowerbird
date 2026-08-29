import { Injectable, inject, signal } from '@angular/core';
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
  readonly submitting = signal(false);
  readonly errorMessage = signal<string | null>(null);

  loadItems(status?: string): void {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.http.listItems(undefined, status).subscribe({
      next: (items) => {
        this.items.set(items);
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => this.handleError(err, 'No se pudieron cargar los ítems.'),
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
    return this.http.listItems(undefined, undefined, query).pipe(
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
