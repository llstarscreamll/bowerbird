import { initFlowbite } from 'flowbite';

import { tap } from 'rxjs';

import { Store } from '@ngrx/store';

import { CommonModule } from '@angular/common';
import { AfterViewInit, Component, OnInit, inject } from '@angular/core';
import { RouterModule } from '@angular/router';

import { getUser } from '@app/ngrx/auth';
import * as finance from '@app/ngrx/finance';
import { FlowbiteService } from '@app/services/flowbite.service';
import { environment } from '@env/environment';

@Component({
  imports: [CommonModule, RouterModule],
  selector: 'app-dashboard-page',
  templateUrl: './dashboard.page.html',
})
export class DashboardPageComponent implements AfterViewInit {
  private store = inject(Store);
  private flowbite = inject(FlowbiteService);

  user$ = this.store.select(getUser);
  selectedWallet$ = this.store
    .select(finance.getSelectedWallet)
    .pipe(tap((w) => (this.selectedWalletID = w?.ID || '')));
  transactions$ = this.store.select(finance.getTransactions);

  selectedWalletID: string = '';
  walletMenuIsOpen = false;
  apiUrl = environment.apiBaseUrl;

  ngAfterViewInit(): void {
    this.flowbite.load(() => initFlowbite());
  }

  syncWalletTransactionsWithEmails() {
    this.store.dispatch(finance.actions.syncTransactionsFromEmail({ walletID: this.selectedWalletID }));
  }
}
