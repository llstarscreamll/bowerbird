import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AuthStore } from '../../../application/auth.store';
import { FormsModule } from '@angular/forms';

import { environment } from '../../../../../environments/environment';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div
      class="flex min-h-screen flex-col items-center justify-center py-10 px-4 sm:px-6 lg:px-8 bg-slate-50/50 dark:bg-slate-950/50"
    >
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
              <label for="email" class="block text-sm font-medium text-slate-700 dark:text-slate-300">
                Correo electronico
              </label>
              <div class="mt-1.5">
                <input
                  id="email"
                  name="email"
                  type="email"
                  autocomplete="email"
                  required
                  [(ngModel)]="email"
                  class="input-field py-2"
                  placeholder="nombre@ejemplo.com"
                />
              </div>
            </div>

            <div>
              <label for="password" class="block text-sm font-medium text-slate-700 dark:text-slate-300">
                Contrasena
              </label>
              <div class="mt-1.5">
                <input
                  id="password"
                  name="password"
                  type="password"
                  autocomplete="current-password"
                  required
                  [(ngModel)]="password"
                  class="input-field py-2"
                  placeholder="••••••••"
                />
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
                    <path
                      d="M1 5L4.5 8.5L13 1"
                      stroke="currentColor"
                      stroke-width="2"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                    />
                  </svg>
                </div>
                <label for="remember-me" class="block text-sm text-slate-600 dark:text-slate-400">Recordarme</label>
              </div>

              <div class="text-sm">
                <a
                  href="#"
                  class="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300 transition-colors"
                >
                  Olvidaste tu contrasena?
                </a>
              </div>
            </div>

            <div
              *ngIf="store.error()"
              class="rounded-md bg-red-50 dark:bg-red-500/10 p-3 border border-red-200 dark:border-red-500/20"
            >
              <div class="flex items-start">
                <div class="flex-shrink-0 mt-0.5">
                  <span class="material-icons-outlined text-red-500 dark:text-red-400 text-sm">error</span>
                </div>
                <div class="ml-2.5">
                  <p class="text-sm font-medium text-red-800 dark:text-red-300">{{ store.error() }}</p>
                </div>
              </div>
            </div>

            <div class="pt-2">
              <button type="submit" [disabled]="store.isLoading()" class="btn-primary w-full py-2.5 shadow-sm text-sm">
                <span *ngIf="!store.isLoading()">Iniciar sesion</span>
                <span *ngIf="store.isLoading()" class="flex items-center justify-center gap-2">
                  <svg class="animate-spin h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path
                      class="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
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
                <span class="bg-white dark:bg-slate-900 px-3 text-[13px] text-slate-500 dark:text-slate-400"
                  >O continua con</span
                >
              </div>
            </div>

            <div class="mt-6 grid grid-cols-2 gap-3">
              <a
                [href]="apiUrl + '/api/v1/auth/google/login'"
                class="btn-secondary w-full gap-2 text-sm py-2 shadow-none border-slate-200 dark:border-slate-700/50"
              >
                <svg class="h-4 w-4" viewBox="0 0 24 24" aria-hidden="true">
                  <path
                    d="M12.0003 4.75C13.7703 4.75 15.3553 5.36002 16.6053 6.54998L20.0303 3.125C17.9502 1.19 15.2353 0 12.0003 0C7.31028 0 3.25527 2.69 1.28027 6.60998L5.27028 9.70498C6.21525 6.86002 8.87028 4.75 12.0003 4.75Z"
                    fill="#EA4335"
                  />
                  <path
                    d="M23.49 12.275C23.49 11.49 23.415 10.73 23.3 10H12V14.51H18.47C18.18 15.99 17.34 17.25 16.08 18.1L19.945 21.1C22.2 19.01 23.49 15.92 23.49 12.275Z"
                    fill="#4285F4"
                  />
                  <path
                    d="M5.26498 14.2949C5.02498 13.5699 4.88501 12.7999 4.88501 11.9999C4.88501 11.1999 5.01998 10.4299 5.26498 9.7049L1.275 6.60986C0.46 8.22986 0 10.0599 0 11.9999C0 13.9399 0.46 15.7699 1.28 17.3899L5.26498 14.2949Z"
                    fill="#FBBC05"
                  />
                  <path
                    d="M12.0004 24.0001C15.2404 24.0001 17.9654 22.935 19.9454 21.095L16.0804 18.095C15.0054 18.82 13.6204 19.245 12.0004 19.245C8.8704 19.245 6.21537 17.135 5.2654 14.29L1.27539 17.385C3.25539 21.31 7.3104 24.0001 12.0004 24.0001Z"
                    fill="#34A853"
                  />
                </svg>
                Google
              </a>
              <a
                [href]="apiUrl + '/api/v1/auth/microsoft/login'"
                class="btn-secondary w-full gap-2 text-sm py-2 shadow-none border-slate-200 dark:border-slate-700/50"
              >
                <svg class="h-4 w-4" viewBox="0 0 21 21" aria-hidden="true">
                  <path d="M10 0H0V10H10V0Z" fill="#F25022" />
                  <path d="M21 0H11V10H21V0Z" fill="#7FBA00" />
                  <path d="M10 11H0V21H10V11Z" fill="#00A4EF" />
                  <path d="M21 11H11V21H21V11Z" fill="#FFB900" />
                </svg>
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
