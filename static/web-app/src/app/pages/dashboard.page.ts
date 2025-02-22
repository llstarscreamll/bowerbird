import { Store } from '@ngrx/store';

import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';

import { getUser } from '@app/ngrx/auth';
import { getSelectedWallet } from '@app/ngrx/finance';
import { environment } from '@env/environment';

@Component({
  imports: [CommonModule],
  selector: 'app-dashboard-page',
  templateUrl: './dashboard.page.html',
})
export class DashboardPageComponent {
  private store = inject(Store);
  user$ = this.store.select(getUser);
  selectedWallet$ = this.store.select(getSelectedWallet);

  walletMenuIsOpen = false;
  apiUrl = environment.apiBaseUrl;
}
