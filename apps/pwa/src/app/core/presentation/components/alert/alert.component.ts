import { Component, computed, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';

export type AlertType = 'info' | 'warning' | 'error' | 'success';

@Component({
  selector: 'app-alert',
  standalone: true,
  imports: [CommonModule],
  template: `
    @if (!isDismissed()) {
      <div [class]="containerClasses()" [attr.role]="role()">
        <div class="flex">
          <div class="shrink-0">
            <span class="material-icons-outlined" [class]="iconClasses()">{{ iconName() }}</span>
          </div>
          <div class="ml-3 w-full">
            @if (title()) {
              <h3 [class]="titleClasses()">{{ title() }}</h3>
            }

            @if (message()) {
              <div class="text-sm" [class]="messageClasses()" [class.mt-2]="title()">
                <p>{{ message() }}</p>
              </div>
            }

            <!-- ng-content for projecting lists or custom HTML -->
            <div class="text-sm" [class]="messageClasses()" [class.mt-2]="title() || message()">
              <ng-content></ng-content>
            </div>
          </div>

          @if (dismissible()) {
            <div class="ml-auto pl-3">
              <div class="-mx-1.5 -my-1.5">
                <button type="button" (click)="dismiss()" [class]="closeButtonClasses()">
                  <span class="sr-only">Cerrar</span>
                  <span class="material-icons-outlined text-base leading-none">close</span>
                </button>
              </div>
            </div>
          }
        </div>
      </div>
    }
  `,
})
export class AlertComponent {
  type = input<AlertType>('info');
  title = input<string>();
  message = input<string>();
  dismissible = input<boolean>(false);

  dismissed = output<void>();

  isDismissed = signal(false);

  role = computed(() => {
    const t = this.type();
    return t === 'error' || t === 'warning' ? 'alert' : 'status';
  });

  containerClasses = computed(() => {
    const base = 'rounded-md p-4 border';
    switch (this.type()) {
      case 'error':
        return `${base} bg-red-50 border-red-200 dark:bg-red-900/20 dark:border-red-800/30`;
      case 'warning':
        return `${base} bg-yellow-50 border-yellow-200 dark:bg-yellow-900/20 dark:border-yellow-800/30`;
      case 'success':
        return `${base} bg-green-50 border-green-200 dark:bg-green-900/20 dark:border-green-800/30`;
      case 'info':
      default:
        return `${base} bg-blue-50 border-blue-200 dark:bg-blue-900/20 dark:border-blue-800/30`;
    }
  });

  iconName = computed(() => {
    switch (this.type()) {
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
  });

  iconClasses = computed(() => {
    switch (this.type()) {
      case 'error':
        return 'text-red-400 dark:text-red-500';
      case 'warning':
        return 'text-yellow-400 dark:text-yellow-500';
      case 'success':
        return 'text-green-400 dark:text-green-500';
      case 'info':
      default:
        return 'text-blue-400 dark:text-blue-500';
    }
  });

  titleClasses = computed(() => {
    const base = 'text-sm font-medium';
    switch (this.type()) {
      case 'error':
        return `${base} text-red-800 dark:text-red-300`;
      case 'warning':
        return `${base} text-yellow-800 dark:text-yellow-300`;
      case 'success':
        return `${base} text-green-800 dark:text-green-300`;
      case 'info':
      default:
        return `${base} text-blue-800 dark:text-blue-300`;
    }
  });

  messageClasses = computed(() => {
    switch (this.type()) {
      case 'error':
        return 'text-red-700 dark:text-red-200';
      case 'warning':
        return 'text-yellow-700 dark:text-yellow-200';
      case 'success':
        return 'text-green-700 dark:text-green-200';
      case 'info':
      default:
        return 'text-blue-700 dark:text-blue-200';
    }
  });

  closeButtonClasses = computed(() => {
    const base = 'inline-flex rounded-md p-1.5 focus:outline-none focus:ring-2 focus:ring-offset-2 transition-colors duration-200';
    switch (this.type()) {
      case 'error':
        return `${base} bg-red-50 text-red-500 hover:bg-red-100 focus:ring-red-600 focus:ring-offset-red-50 dark:bg-transparent dark:text-red-400 dark:hover:bg-red-900/50`;
      case 'warning':
        return `${base} bg-yellow-50 text-yellow-500 hover:bg-yellow-100 focus:ring-yellow-600 focus:ring-offset-yellow-50 dark:bg-transparent dark:text-yellow-400 dark:hover:bg-yellow-900/50`;
      case 'success':
        return `${base} bg-green-50 text-green-500 hover:bg-green-100 focus:ring-green-600 focus:ring-offset-green-50 dark:bg-transparent dark:text-green-400 dark:hover:bg-green-900/50`;
      case 'info':
      default:
        return `${base} bg-blue-50 text-blue-500 hover:bg-blue-100 focus:ring-blue-600 focus:ring-offset-blue-50 dark:bg-transparent dark:text-blue-400 dark:hover:bg-blue-900/50`;
    }
  });

  dismiss() {
    this.isDismissed.set(true);
    this.dismissed.emit();
  }
}
