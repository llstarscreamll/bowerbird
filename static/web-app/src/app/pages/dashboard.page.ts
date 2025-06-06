import { initFlowbite } from 'flowbite';

import { Observable, filter, tap } from 'rxjs';

import { Store } from '@ngrx/store';

import { CommonModule } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { RouterModule } from '@angular/router';

import { getUser } from '@app/ngrx/auth';
import * as finance from '@app/ngrx/finance';
import { FlowbiteService } from '@app/services/flowbite.service';
import { Wallet } from '@app/types';
import { environment } from '@env/environment';

@Component({
  imports: [CommonModule, RouterModule],
  selector: 'app-dashboard-page',
  templateUrl: './dashboard.page.html',
})
export class DashboardPageComponent implements OnInit {
  private store = inject(Store);
  private flowbite = inject(FlowbiteService);

  user$ = this.store.select(getUser);
  selectedWallet$!: Observable<Wallet | null>;
  loading$ = this.store.select(finance.getLoading);
  metrics$ = this.store.select(finance.getMetrics);
  transactions$ = this.store.select(finance.getTransactions);

  currentDate = new Date();
  selectedWalletID: string = '';
  apiUrl = environment.apiBaseUrl;

  ngOnInit(): void {
    this.store.dispatch(finance.actions.getWallets());

    this.selectedWallet$ = this.store.select(finance.getSelectedWallet).pipe(
      filter((w): w is Wallet => w !== null),
      tap((w) => (this.selectedWalletID = w.ID)),
      tap(() => this.store.dispatch(finance.actions.getTransactions({ walletID: this.selectedWalletID }))),
      tap(() =>
        this.store.dispatch(
          finance.actions.getMetrics({
            walletID: this.selectedWalletID,
            from: new Date(this.currentDate.getFullYear(), this.currentDate.getMonth(), 1),
            to: new Date(this.currentDate.getFullYear(), this.currentDate.getMonth() + 1, 0),
          }),
        ),
      ),
      tap(() => this.flowbite.load(() => initFlowbite())),
    );
  }

  syncWalletTransactionsWithEmails() {
    this.store.dispatch(finance.actions.syncTransactionsFromEmail({ walletID: this.selectedWalletID }));
  }
}
