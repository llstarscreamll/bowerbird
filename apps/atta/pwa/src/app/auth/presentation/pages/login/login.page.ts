import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AuthStore } from '../../../application/auth.store';
import { FormsModule } from '@angular/forms';
import { NgIcon } from '@ng-icons/core';
import { environment } from '../../../../../environments/environment';
import { IconGoogleComponent } from '../../../../core/presentation/components/icons/icon-google.component';
import { IconMicrosoftComponent } from '../../../../core/presentation/components/icons/icon-microsoft.component';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmCheckboxImports } from '@spartan-ng/helm/checkbox';
import { HlmFieldImports } from '@spartan-ng/helm/field';
import { HlmInputImports } from '@spartan-ng/helm/input';
import { HlmLabelImports } from '@spartan-ng/helm/label';
import { HlmSeparatorImports } from '@spartan-ng/helm/separator';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    NgIcon,
    IconGoogleComponent,
    IconMicrosoftComponent,
    HlmCardImports,
    HlmFieldImports,
    HlmInputImports,
    HlmLabelImports,
    HlmButtonImports,
    HlmAlertImports,
    HlmCheckboxImports,
    HlmSeparatorImports,
    HlmSpinnerImports,
  ],
  template: `
    <div class="flex min-h-screen flex-col items-center justify-center bg-muted/30 px-4 py-10 sm:px-6 lg:px-8">
      <div class="mb-6 text-center sm:mx-auto sm:w-full sm:max-w-[400px]">
        <div class="mx-auto flex size-12 items-center justify-center rounded-xl bg-primary shadow-sm">
          <ng-icon name="lucidePlaneTakeoff" class="text-xl text-primary-foreground" />
        </div>
        <h2 class="mt-6 text-2xl font-semibold tracking-tight">Inicia sesion</h2>
        <p class="mt-2 text-sm text-muted-foreground">Hola de nuevo. Por favor, ingresa tus datos.</p>
      </div>

      <hlm-card class="w-full max-w-[400px] p-6 sm:p-8">
        <form class="space-y-5" (ngSubmit)="onLocalLogin()">
          <hlm-field>
            <label hlmLabel for="email">Correo electronico</label>
            <input hlmInput id="email" name="email" type="email" autocomplete="email" required [(ngModel)]="email" placeholder="nombre@ejemplo.com" />
          </hlm-field>

          <hlm-field>
            <label hlmLabel for="password">Contrasena</label>
            <input hlmInput id="password" name="password" type="password" autocomplete="current-password" required [(ngModel)]="password" placeholder="••••••••" />
          </hlm-field>

          <div class="flex items-center justify-between pt-1">
            <div class="flex items-center gap-2.5">
              <hlm-checkbox id="remember-me" name="remember-me" />
              <label hlmLabel for="remember-me" class="font-normal text-muted-foreground">Recordarme</label>
            </div>
            <a href="#" class="text-sm font-medium text-primary hover:underline">Olvidaste tu contrasena?</a>
          </div>

          @if (store.error()) {
            <hlm-alert variant="destructive">
              <ng-icon hlm name="lucideCircleAlert" />
              <h4 hlmAlertTitle>Error</h4>
              <p hlmAlertDescription>{{ store.error() }}</p>
            </hlm-alert>
          }

          <button type="submit" hlmBtn class="w-full" [disabled]="store.isLoading()">
            @if (store.isLoading()) {
              <hlm-spinner class="size-4" />
              Iniciando sesion...
            } @else {
              Iniciar sesion
            }
          </button>
        </form>

        <div class="relative my-6">
          <hlm-separator />
          <span class="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 bg-card px-3 text-[13px] text-muted-foreground">O continua con</span>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <a hlmBtn variant="outline" class="w-full" [href]="apiUrl + '/api/v1/auth/google/login'">
            <app-icon-google class="size-4" />
            Google
          </a>
          <a hlmBtn variant="outline" class="w-full" [href]="apiUrl + '/api/v1/auth/microsoft/login'">
            <app-icon-microsoft class="size-4" />
            Microsoft
          </a>
        </div>
      </hlm-card>
    </div>
  `,
})
export class LoginPage {
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
