import { Subject, takeUntil, tap } from 'rxjs';

import { Store } from '@ngrx/store';

import { CommonModule } from '@angular/common';
import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { ActivatedRoute, RouterModule } from '@angular/router';

import { getTransaction } from '@app/ngrx/finance';
import { actions } from '@app/ngrx/finance';

@Component({
  selector: 'app-transaction-detail',
  templateUrl: './transaction-detail.page.html',
  imports: [CommonModule, RouterModule],
})
export class TransactionDetailPage implements OnInit, OnDestroy {
  private store = inject(Store);
  private route = inject(ActivatedRoute);

  walletID = this.route.snapshot.params['walletID'];
  transactionID = this.route.snapshot.params['transactionID'];

  transaction$ = this.store.select(getTransaction);

  constructor() {}

  ngOnInit() {
    this.store.dispatch(actions.getTransaction({ walletID: this.walletID, transactionID: this.transactionID }));
  }

  ngOnDestroy() {
    this.store.dispatch(actions.setSelectedTransaction({ transaction: null }));
  }
}
