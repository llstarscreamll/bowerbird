import { Injectable, inject, signal } from '@angular/core';
import { finalize } from 'rxjs';
import { Secret, SecretPurpose } from '../domain/secrets.model';
import { SecretsHttpRepository } from '../infrastructure/secrets.http.repository';
import { ToastService } from '../../core/services/toast.service';

@Injectable({ providedIn: 'root' })
export class SecretsStore {
  readonly secrets = signal<Secret[]>([]);
  readonly loading = signal(false);
  readonly submitting = signal(false);
  readonly errorMessage = signal('');

  private readonly repository = inject(SecretsHttpRepository);
  private readonly toast = inject(ToastService);

  loadSecrets(): void {
    this.loading.set(true);
    this.errorMessage.set('');
    this.repository
      .list()
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe({
        next: (secrets) => this.secrets.set(secrets),
        error: () => this.errorMessage.set('No fue posible cargar las credenciales.'),
      });
  }

  createSecret(input: { purpose: SecretPurpose; label: string; value: string }): void {
    this.submitting.set(true);
    this.errorMessage.set('');
    this.repository
      .create(input)
      .pipe(finalize(() => this.submitting.set(false)))
      .subscribe({
        next: (secret) => {
          this.secrets.update((list) => [...list, secret].sort((a, b) => a.label.localeCompare(b.label)));
          this.toast.showSuccess('Credencial guardada');
        },
        error: () => this.errorMessage.set('No fue posible guardar la credencial.'),
      });
  }

  rotateSecret(id: string, value: string): void {
    this.submitting.set(true);
    this.errorMessage.set('');
    this.repository
      .update(id, { value })
      .pipe(finalize(() => this.submitting.set(false)))
      .subscribe({
        next: (secret) => {
          this.secrets.update((list) => list.map((item) => (item.id === id ? secret : item)));
          this.toast.showSuccess('Credencial actualizada');
        },
        error: () => this.errorMessage.set('No fue posible actualizar la credencial.'),
      });
  }

  deleteSecret(id: string): void {
    this.submitting.set(true);
    this.errorMessage.set('');
    this.repository
      .delete(id)
      .pipe(finalize(() => this.submitting.set(false)))
      .subscribe({
        next: () => {
          this.secrets.update((list) => list.filter((item) => item.id !== id));
          this.toast.showSuccess('Credencial eliminada');
        },
        error: () => this.errorMessage.set('No fue posible eliminar la credencial.'),
      });
  }
}
