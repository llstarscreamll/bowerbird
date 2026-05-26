import { Injectable, signal } from '@angular/core';

export type ToastType = 'info' | 'warning' | 'error' | 'success';

export interface Toast {
  id: string;
  type: ToastType;
  title?: string;
  message: string;
  duration?: number;
}

@Injectable({
  providedIn: 'root',
})
export class ToastService {
  private toastsSignal = signal<Toast[]>([]);
  public readonly toasts = this.toastsSignal.asReadonly();

  show(toast: Omit<Toast, 'id'>) {
    const id = Math.random().toString(36).substring(2, 9);
    const newToast = { ...toast, id };

    this.toastsSignal.update((toasts) => [...toasts, newToast]);

    if (toast.duration !== 0) {
      setTimeout(() => this.remove(id), toast.duration || 5000);
    }
  }

  showSuccess(message: string, title?: string) {
    this.show({ type: 'success', message, title });
  }

  showError(message: string, title?: string, duration: number = 7000) {
    this.show({ type: 'error', message, title, duration });
  }

  showInfo(message: string, title?: string) {
    this.show({ type: 'info', message, title });
  }

  showWarning(message: string, title?: string) {
    this.show({ type: 'warning', message, title });
  }

  remove(id: string) {
    this.toastsSignal.update((toasts) => toasts.filter((t) => t.id !== id));
  }
}
