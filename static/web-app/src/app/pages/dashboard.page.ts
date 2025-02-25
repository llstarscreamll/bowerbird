import { tap } from 'rxjs';

import { Store } from '@ngrx/store';

import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';

import { getUser } from '@app/ngrx/auth';
import * as finance from '@app/ngrx/finance';
import { environment } from '@env/environment';

@Component({
  imports: [CommonModule],
  selector: 'app-dashboard-page',
  templateUrl: './dashboard.page.html',
})
export class DashboardPageComponent {
  private store = inject(Store);
  user$ = this.store.select(getUser);
  selectedWallet$ = this.store
    .select(finance.getSelectedWallet)
    .pipe(tap((w) => (this.selectedWalletID = w?.ID || '')));
  transactions$ = this.store.select(finance.getTransactions);

  selectedWalletID: string = '';
  walletMenuIsOpen = false;
  apiUrl = environment.apiBaseUrl;

  syncWalletTransactionsWithEmails() {
    this.store.dispatch(finance.actions.syncTransactionsFromEmail({ walletID: this.selectedWalletID }));
  }
}
