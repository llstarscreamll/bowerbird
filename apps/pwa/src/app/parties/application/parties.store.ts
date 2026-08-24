import { Injectable, inject, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { ToastService } from '../../core/services/toast.service';
import { PartiesHttpService } from '../infrastructure/parties.http.service';
import { Party } from '../domain/party.model';

@Injectable({ providedIn: 'root' })
export class PartiesStore {
  private readonly http = inject(PartiesHttpService);
  private readonly toast = inject(ToastService);

  readonly parties = signal<Party[]>([]);
  readonly loading = signal(false);
  readonly errorMessage = signal<string | null>(null);

  load(role?: string): void {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.http.list(role).subscribe({
      next: (parties) => {
        this.parties.set(parties);
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => {
        this.loading.set(false);
        if (err.status >= 400 && err.status < 500) {
          this.errorMessage.set(err.error?.errors?.[0]?.detail || 'No se pudieron cargar las contrapartes.');
        } else {
          this.toast.showError('Error al cargar contrapartes.');
        }
      },
    });
  }
}
