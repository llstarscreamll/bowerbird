import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AuthStore } from '../../../application/auth.store';
import { FormsModule } from '@angular/forms';

import { environment } from '../../../../../environments/environment';

import { AlertComponent } from '../../../../core/presentation/components/alert/alert.component';
import { IconGoogleComponent } from '../../../../core/presentation/components/icons/icon-google.component';
import { IconMicrosoftComponent } from '../../../../core/presentation/components/icons/icon-microsoft.component';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, FormsModule, AlertComponent, IconGoogleComponent, IconMicrosoftComponent],
  template: `
    <div class="flex min-h-screen flex-col items-center justify-center py-10 px-4 sm:px-6 lg:px-8 bg-slate-50/50 dark:bg-slate-950/50">
      <div class="sm:mx-auto sm:w-full sm:max-w-[400px] text-center mb-6">
        <div class="mx-auto h-12 w-12 bg-indigo-600 rounded-xl flex items-center justify-center shadow-sm">
          <span class="material-icons-outlined text-white text-2xl">flight_takeoff</span>
        </div>
        <h2 class="mt-6 text-2xl font-semibold tracking-tight text-slate-900 dark:text-white">Inicia sesion</h2>
        <p class="mt-2 text-sm text-slate-500 dark:text-slate-400">Hola de nuevo. Por favor, ingresa tus datos.</p>
      </div>

      <div class="sm:mx-auto sm:w-full sm:max-w-[400px]">
        <div class="card p-6 sm:p-8 space-y-6 shadow-sm">
          <form class="space-y-5" (ngSubmit)="onLocalLogin()">
            <div>
              <label for="email" class="block text-sm font-medium text-slate-700 dark:text-slate-300"> Correo electronico </label>
              <div class="mt-1.5 relative">
                <input
                  id="email"
                  name="email"
                  type="email"
                  autocomplete="email"
                  required
                  aria-errormessage="email-error"
                  [(ngModel)]="email"
                  class="input-field py-2"
                  placeholder="nombre@ejemplo.com"
                />
                <div id="email-error" class="input-error-msg flex items-center gap-1">
                  <span class="material-icons-outlined text-[16px]" aria-hidden="true">error_outline</span>
                  <span>Este campo es requerido.</span>
                </div>
              </div>
            </div>

            <div>
              <label for="password" class="block text-sm font-medium text-slate-700 dark:text-slate-300"> Contrasena </label>
              <div class="mt-1.5 relative">
                <input
                  id="password"
                  name="password"
                  type="password"
                  autocomplete="current-password"
                  required
                  aria-errormessage="password-error"
                  [(ngModel)]="password"
                  class="input-field py-2"
                  placeholder="••••••••"
                />
                <div id="password-error" class="input-error-msg flex items-center gap-1">
                  <span class="material-icons-outlined text-[16px]" aria-hidden="true">error_outline</span>
                  <span>Este campo es requerido.</span>
                </div>
              </div>
            </div>

            <div class="flex items-center justify-between pt-1">
              <div class="flex items-center gap-2.5">
                <div class="relative flex h-5 items-center">
                  <input
                    id="remember-me"
                    name="remember-me"
                    type="checkbox"
                    class="peer appearance-none h-4 w-4 rounded border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 checked:border-indigo-600 dark:checked:border-indigo-500 checked:bg-indigo-600 dark:checked:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-600 dark:focus:ring-indigo-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900 transition-colors"
                  />
                  <svg
                    class="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-2.5 h-2.5 text-white opacity-0 peer-checked:opacity-100 pointer-events-none"
                    viewBox="0 0 14 10"
                    fill="none"
                  >
                    <path d="M1 5L4.5 8.5L13 1" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </div>
                <label for="remember-me" class="block text-sm text-slate-600 dark:text-slate-400">Recordarme</label>
              </div>

              <div class="text-sm">
                <a href="#" class="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300 transition-colors"> Olvidaste tu contrasena? </a>
              </div>
            </div>

            <app-alert *ngIf="store.error()" type="error">
              {{ store.error() }}
            </app-alert>

            <div class="pt-2">
              <button type="submit" [disabled]="store.isLoading()" class="btn-primary w-full py-2.5 shadow-sm text-sm">
                <span *ngIf="!store.isLoading()">Iniciar sesion</span>
                <span *ngIf="store.isLoading()" class="flex items-center justify-center gap-2">
                  <svg class="animate-spin h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  Iniciando sesion...
                </span>
              </button>
            </div>
          </form>

          <div class="pt-2 pb-1">
            <div class="relative">
              <div class="absolute inset-0 flex items-center" aria-hidden="true">
                <div class="w-full border-t border-slate-200 dark:border-slate-800/80"></div>
              </div>
              <div class="relative flex justify-center">
                <span class="bg-white dark:bg-slate-900 px-3 text-[13px] text-slate-500 dark:text-slate-400">O continua con</span>
              </div>
            </div>

            <div class="mt-6 grid grid-cols-2 gap-3">
              <a [href]="apiUrl + '/api/v1/auth/google/login'" class="btn-secondary w-full gap-2 text-sm py-2 shadow-none border-slate-200 dark:border-slate-700/50">
                <div class="h-4 w-4">
                  <app-icon-google></app-icon-google>
                </div>
                Google
              </a>
              <a [href]="apiUrl + '/api/v1/auth/microsoft/login'" class="btn-secondary w-full gap-2 text-sm py-2 shadow-none border-slate-200 dark:border-slate-700/50">
                <div class="h-4 w-4">
                  <app-icon-microsoft></app-icon-microsoft>
                </div>
                Microsoft
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  `,
})
export class LoginComponent {
  readonly store = inject(AuthStore);
  private router = inject(Router);

  apiUrl = environment.apiUrl;

  email = '';
  password = '';

  constructor() {
    if (this.store.isAuthenticated()) {
      this.router.navigate(['/lobby']);
    }
  }

  onLocalLogin() {
    if (this.email && this.password) {
      this.store.loginLocal({
        email: this.email,
        password: this.password,
        onSuccess: () => {
          this.router.navigate(['/lobby']);
        },
      });
    }
  }
}
