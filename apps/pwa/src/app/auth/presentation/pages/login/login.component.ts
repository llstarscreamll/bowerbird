import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AuthStore } from '../../../application/auth.store';
import { FormsModule } from '@angular/forms';
import { AuthHttpService } from '../../../infrastructure/auth.http.service';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
      <div class="max-w-md w-full space-y-8">
        <div>
          <h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">Sign in to your account</h2>
        </div>

        <!-- Local Login (Dev only) -->
        <form class="mt-8 space-y-6" (ngSubmit)="onLocalLogin()">
          <div class="rounded-md shadow-sm -space-y-px">
            <div>
              <label for="email-address" class="sr-only">Email address</label>
              <input
                id="email-address"
                name="email"
                type="email"
                autocomplete="email"
                required
                [(ngModel)]="email"
                class="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-t-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
                placeholder="Email address"
              />
            </div>
            <div>
              <label for="password" class="sr-only">Password</label>
              <input
                id="password"
                name="password"
                type="password"
                autocomplete="current-password"
                required
                [(ngModel)]="password"
                class="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-b-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
                placeholder="Password"
              />
            </div>
          </div>

          <div class="flex items-center justify-between">
            <div class="text-sm">
              <span class="text-red-500" *ngIf="store.error()">{{ store.error() }}</span>
            </div>
          </div>

          <div>
            <button
              type="submit"
              [disabled]="store.isLoading()"
              class="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
            >
              <span *ngIf="!store.isLoading()">Sign in (Local)</span>
              <span *ngIf="store.isLoading()">Signing in...</span>
            </button>
          </div>
        </form>

        <div class="mt-6">
          <div class="relative">
            <div class="absolute inset-0 flex items-center">
              <div class="w-full border-t border-gray-300"></div>
            </div>
            <div class="relative flex justify-center text-sm">
              <span class="px-2 bg-gray-50 text-gray-500"> Or continue with </span>
            </div>
          </div>

          <div class="mt-6 grid grid-cols-2 gap-3">
            <div>
              <a
                href="http://api.bowerbird.dev/api/v1/auth/google/login"
                class="w-full inline-flex justify-center py-2 px-4 border border-gray-300 rounded-md shadow-sm bg-white text-sm font-medium text-gray-500 hover:bg-gray-50"
              >
                <span>Google</span>
              </a>
            </div>
            <div>
              <a
                href="http://api.bowerbird.dev/api/v1/auth/microsoft/login"
                class="w-full inline-flex justify-center py-2 px-4 border border-gray-300 rounded-md shadow-sm bg-white text-sm font-medium text-gray-500 hover:bg-gray-50"
              >
                <span>Microsoft</span>
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
  private authHttp = inject(AuthHttpService);

  email = '';
  password = '';

  constructor() {
    // If we have a token, go to lobby
    if (this.store.isAuthenticated()) {
      this.router.navigate(['/lobby']);
    }
  }

  onLocalLogin() {
    if (this.email && this.password) {
      // Small hack: instead of using rxMethod which we'd need to subscribe to,
      // let's just do it directly or subscribe to a side effect.
      // Since it's a SignalStore, the state updates automatically.
      this.store.loginLocal({ email: this.email, password: this.password });

      // Navigate on next tick if authenticated
      setTimeout(() => {
        if (this.store.isAuthenticated()) {
          this.router.navigate(['/lobby']);
        }
      }, 500);
    }
  }
}
