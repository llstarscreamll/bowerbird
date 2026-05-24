import { Component, inject, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthStore } from '../../../application/auth.store';

@Component({
  selector: 'app-auth-callback',
  standalone: true,
  template: `
    <div class="flex min-h-screen flex-col items-center justify-center py-12">
      <div class="flex flex-col items-center space-y-4">
        <svg class="animate-spin h-8 w-8 text-indigo-600" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path
            class="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          ></path>
        </svg>
        <p class="text-slate-600 dark:text-slate-400">Authenticating...</p>
      </div>
    </div>
  `,
})
export class CallbackComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private store = inject(AuthStore);

  ngOnInit() {
    this.route.queryParams.subscribe((params) => {
      const token = params['access_token'];
      if (token) {
        this.store.setToken(token);
        this.router.navigate(['/lobby']);
      } else {
        // Handle error or missing token
        this.router.navigate(['/login']);
      }
    });
  }
}
