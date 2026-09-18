import { HttpErrorResponse } from '@angular/common/http';
import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, catchError, of, tap } from 'rxjs';
import { ToastService } from '../../core/services/toast.service';
import { CreateLegalEntityInput, LegalEntity, UpdateLegalEntityInput } from '../domain/legal-entity.model';
import { LegalEntitiesHttpService } from '../infrastructure/legal-entities.http.service';

@Injectable({ providedIn: 'root' })
export class LegalEntitiesStore {
  private readonly http = inject(LegalEntitiesHttpService);
  private readonly toast = inject(ToastService);

  readonly entities = signal<LegalEntity[]>([]);
  readonly loading = signal(false);
  readonly submitting = signal(false);
  readonly errorMessage = signal<string | null>(null);

  readonly current = computed(() => this.entities()[0] ?? null);
  readonly hasAny = computed(() => this.entities().length > 0);

  load(): void {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.http.list().subscribe({
      next: (items) => {
        this.entities.set(items);
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => this.handleError(err, 'No se pudo cargar la identificación tributaria.'),
    });
  }

  create(input: CreateLegalEntityInput): Observable<LegalEntity | null> {
    this.submitting.set(true);
    this.errorMessage.set(null);
    return this.http.create(input).pipe(
      tap((entity) => {
        this.submitting.set(false);
        this.entities.set([entity]);
        this.toast.showSuccess('Identificación registrada. Revisaremos el correo existente para extraer facturas a tu nombre.');
      }),
      catchError((err: HttpErrorResponse) => {
        this.submitting.set(false);
        this.handleError(err, 'No se pudo registrar la identificación tributaria.');
        return of(null);
      }),
    );
  }

  update(id: string, input: UpdateLegalEntityInput): Observable<LegalEntity | null> {
    this.submitting.set(true);
    this.errorMessage.set(null);
    return this.http.update(id, input).pipe(
      tap((entity) => {
        this.submitting.set(false);
        this.entities.set([entity]);
        this.toast.showSuccess('Identificación actualizada.');
      }),
      catchError((err: HttpErrorResponse) => {
        this.submitting.set(false);
        this.handleError(err, 'No se pudo actualizar la identificación tributaria.');
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
