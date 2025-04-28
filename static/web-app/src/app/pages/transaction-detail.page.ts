import { initFlowbite } from 'flowbite';

import { tap } from 'rxjs';

import { Store } from '@ngrx/store';

import { CommonModule } from '@angular/common';
import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { ActivatedRoute, RouterModule } from '@angular/router';

import { getCategories, getTransaction } from '@app/ngrx/finance';
import { actions } from '@app/ngrx/finance';
import { FlowbiteService } from '@app/services/flowbite.service';

@Component({
  selector: 'app-transaction-detail',
  templateUrl: './transaction-detail.page.html',
  imports: [CommonModule, RouterModule],
})
export class TransactionDetailPage implements OnInit, OnDestroy {
  private store = inject(Store);
  private route = inject(ActivatedRoute);
  private flowbite = inject(FlowbiteService);

  walletID = this.route.snapshot.params['walletID'];
  transactionID = this.route.snapshot.params['transactionID'];

  categories$ = this.store.select(getCategories);
  transaction$ = this.store.select(getTransaction).pipe(tap(() => this.flowbite.load(() => initFlowbite())));
  constructor() {}

  ngOnInit() {
    this.store.dispatch(actions.getTransaction({ walletID: this.walletID, transactionID: this.transactionID }));
    this.store.dispatch(actions.getCategories({ walletID: this.walletID }));
  }

  ngOnDestroy() {
    this.store.dispatch(actions.setSelectedTransaction({ transaction: null }));
  }
}
