import { Injectable } from '@angular/core';
import { toast } from '@spartan-ng/brain/sonner';

export type ToastType = 'info' | 'warning' | 'error' | 'success';

interface ToastPayload {
  type: ToastType;
  title?: string;
  message: string;
  duration?: number;
}

@Injectable({
  providedIn: 'root',
})
export class ToastService {
  show(toastPayload: ToastPayload) {
    const { type, message, title, duration } = toastPayload;
    const options = {
      description: title ? message : undefined,
      duration: duration ?? (type === 'error' ? 7000 : 5000),
    };
    const content = title ?? message;

    switch (type) {
      case 'success':
        toast.success(content, options);
        break;
      case 'error':
        toast.error(content, options);
        break;
      case 'warning':
        toast.warning(content, options);
        break;
      case 'info':
      default:
        toast.info(content, options);
        break;
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
}
