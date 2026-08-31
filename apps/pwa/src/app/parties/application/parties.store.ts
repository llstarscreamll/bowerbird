import { Injectable, inject, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable, catchError, of, tap } from 'rxjs';
import { ToastService } from '../../core/services/toast.service';
import { PartiesHttpService } from '../infrastructure/parties.http.service';
import { CreatePartyInput, Party, UpdatePartyInput } from '../domain/party.model';

@Injectable({ providedIn: 'root' })
export class PartiesStore {
  private readonly http = inject(PartiesHttpService);
  private readonly toast = inject(ToastService);

  readonly parties = signal<Party[]>([]);
  readonly selectedParty = signal<Party | null>(null);
  readonly loading = signal(false);
  readonly submitting = signal(false);
  readonly errorMessage = signal<string | null>(null);

  load(role?: string): void {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.http.list(role).subscribe({
      next: (parties) => {
        this.parties.set(parties);
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => this.handleError(err, 'No se pudieron cargar los contactos.'),
    });
  }

  loadParty(id: string): void {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.selectedParty.set(null);
    this.http.getParty(id).subscribe({
      next: (party) => {
        this.selectedParty.set(party);
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => this.handleError(err, 'No se pudo cargar el contacto.'),
    });
  }

  createParty(input: CreatePartyInput): Observable<Party | null> {
    this.submitting.set(true);
    this.errorMessage.set(null);
    return this.http.createParty(input).pipe(
      tap((party) => {
        this.submitting.set(false);
        this.selectedParty.set(party);
        this.toast.showSuccess('Contacto guardado exitosamente.');
      }),
      catchError((err: HttpErrorResponse) => {
        this.submitting.set(false);
        this.handleError(err, 'No se pudo crear el contacto.');
        return of(null);
      }),
    );
  }

  updateParty(id: string, input: UpdatePartyInput): Observable<Party | null> {
    this.submitting.set(true);
    this.errorMessage.set(null);
    return this.http.updateParty(id, input).pipe(
      tap((party) => {
        this.submitting.set(false);
        this.selectedParty.set(party);
        this.toast.showSuccess('Contacto guardado exitosamente.');
      }),
      catchError((err: HttpErrorResponse) => {
        this.submitting.set(false);
        this.handleError(err, 'No se pudo actualizar el contacto.');
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
