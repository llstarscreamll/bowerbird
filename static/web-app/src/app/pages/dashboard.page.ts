import { initFlowbite } from 'flowbite';

import { Observable, filter, take, takeLast, tap } from 'rxjs';

import { Store } from '@ngrx/store';

import { CommonModule } from '@angular/common';
import { AfterViewInit, Component, OnInit, inject } from '@angular/core';
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
export class DashboardPageComponent implements AfterViewInit, OnInit {
  private store = inject(Store);
  private flowbite = inject(FlowbiteService);

  user$ = this.store.select(getUser);
  selectedWallet$!: Observable<Wallet | null>;
  transactions$ = this.store.select(finance.getTransactions);

  selectedWalletID: string = '';
  apiUrl = environment.apiBaseUrl;

  ngOnInit(): void {
    this.store.dispatch(finance.actions.getWallets());

    this.selectedWallet$ = this.store.select(finance.getSelectedWallet).pipe(
      filter((w): w is Wallet => w !== null),
      tap((w) => (this.selectedWalletID = w.ID)),
      tap(() => this.store.dispatch(finance.actions.getTransactions({ walletID: this.selectedWalletID }))),
    );
  }

  ngAfterViewInit(): void {
    this.flowbite.load(() => initFlowbite());
  }

  syncWalletTransactionsWithEmails() {
    this.store.dispatch(finance.actions.syncTransactionsFromEmail({ walletID: this.selectedWalletID }));
  }
}
