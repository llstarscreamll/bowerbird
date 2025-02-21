import { Store } from '@ngrx/store';
import { RouterLink } from '@angular/router';
import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';

import * as auth from '../ngrx/auth';
import { environment } from '../../environments/environment';

@Component({
  imports: [CommonModule, RouterLink],
  selector: 'app-landing-page',
  templateUrl: './landing.page.html',
})
export class LandingPageComponent {
  store = inject(Store);
  loggedIn$ = this.store.select(auth.getLoggedIn);

  apiBaseURl = environment.apiBaseUrl;
}
