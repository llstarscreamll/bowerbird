import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ToastService, Toast, ToastType } from '../../../services/toast.service';

@Component({
  selector: 'app-toast-container',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2 pointer-events-none">
      @for (toast of toastService.toasts(); track toast.id) {
        <div class="pointer-events-auto min-w-[300px] max-w-sm rounded-lg border shadow-lg overflow-hidden transition-all duration-300" [ngClass]="getContainerClasses(toast.type)" role="alert">
          <div class="p-4 flex items-start">
            <div class="shrink-0">
              <span class="material-icons-outlined" [ngClass]="getIconClasses(toast.type)">
                {{ getIconName(toast.type) }}
              </span>
            </div>
            <div class="ml-3 w-0 flex-1 pt-0.5">
              @if (toast.title) {
                <p class="text-sm font-medium" [ngClass]="getTitleClasses(toast.type)">
                  {{ toast.title }}
                </p>
              }
              <p class="text-sm" [ngClass]="getMessageClasses(toast.type)" [class.mt-1]="toast.title">
                {{ toast.message }}
              </p>
            </div>
            <div class="ml-4 flex shrink-0">
              <button
                type="button"
                (click)="toastService.remove(toast.id)"
                class="inline-flex rounded-md p-1.5 focus:outline-none focus:ring-2 focus:ring-offset-2 transition-colors duration-200"
                [ngClass]="getCloseButtonClasses(toast.type)"
              >
                <span class="sr-only">Cerrar</span>
                <span class="material-icons-outlined text-base leading-none">close</span>
              </button>
            </div>
          </div>
        </div>
      }
    </div>
  `,
})
export class ToastContainerComponent {
  public toastService = inject(ToastService);

  getContainerClasses(type: ToastType): string {
    switch (type) {
      case 'error':
        return 'bg-red-50 border-red-200 dark:bg-red-900/90 dark:border-red-800/50';
      case 'warning':
        return 'bg-yellow-50 border-yellow-200 dark:bg-yellow-900/90 dark:border-yellow-800/50';
      case 'success':
        return 'bg-green-50 border-green-200 dark:bg-green-900/90 dark:border-green-800/50';
      case 'info':
      default:
        return 'bg-blue-50 border-blue-200 dark:bg-blue-900/90 dark:border-blue-800/50';
    }
  }

  getIconName(type: ToastType): string {
    switch (type) {
      case 'error':
        return 'error_outline';
      case 'warning':
        return 'warning_amber';
      case 'success':
        return 'check_circle_outline';
      case 'info':
      default:
        return 'info';
    }
  }

  getIconClasses(type: ToastType): string {
    switch (type) {
      case 'error':
        return 'text-red-400 dark:text-red-400';
      case 'warning':
        return 'text-yellow-400 dark:text-yellow-400';
      case 'success':
        return 'text-green-400 dark:text-green-400';
      case 'info':
      default:
        return 'text-blue-400 dark:text-blue-400';
    }
  }

  getTitleClasses(type: ToastType): string {
    switch (type) {
      case 'error':
        return 'text-red-800 dark:text-red-200';
      case 'warning':
        return 'text-yellow-800 dark:text-yellow-200';
      case 'success':
        return 'text-green-800 dark:text-green-200';
      case 'info':
      default:
        return 'text-blue-800 dark:text-blue-200';
    }
  }

  getMessageClasses(type: ToastType): string {
    switch (type) {
      case 'error':
        return 'text-red-700 dark:text-red-300';
      case 'warning':
        return 'text-yellow-700 dark:text-yellow-300';
      case 'success':
        return 'text-green-700 dark:text-green-300';
      case 'info':
      default:
        return 'text-blue-700 dark:text-blue-300';
    }
  }

  getCloseButtonClasses(type: ToastType): string {
    switch (type) {
      case 'error':
        return 'bg-transparent text-red-500 hover:bg-red-100 focus:ring-red-600 focus:ring-offset-red-50 dark:text-red-400 dark:hover:bg-red-900/50';
      case 'warning':
        return 'bg-transparent text-yellow-500 hover:bg-yellow-100 focus:ring-yellow-600 focus:ring-offset-yellow-50 dark:text-yellow-400 dark:hover:bg-yellow-900/50';
      case 'success':
        return 'bg-transparent text-green-500 hover:bg-green-100 focus:ring-green-600 focus:ring-offset-green-50 dark:text-green-400 dark:hover:bg-green-900/50';
      case 'info':
      default:
        return 'bg-transparent text-blue-500 hover:bg-blue-100 focus:ring-blue-600 focus:ring-offset-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/50';
    }
  }
}
