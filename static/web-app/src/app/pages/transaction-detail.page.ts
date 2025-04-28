import { Dropdown, initFlowbite } from 'flowbite';

import { Subject, debounceTime, filter, takeUntil, tap } from 'rxjs';

import { Store } from '@ngrx/store';

import { CommonModule } from '@angular/common';
import {
  AfterViewInit,
  Component,
  ElementRef,
  OnChanges,
  OnDestroy,
  OnInit,
  SimpleChanges,
  ViewChild,
  inject,
} from '@angular/core';
import { ActivatedRoute, RouterModule } from '@angular/router';

import { getCategories, getSelectedTransaction } from '@app/ngrx/finance';
import { actions } from '@app/ngrx/finance';
import { FlowbiteService } from '@app/services/flowbite.service';
import { Transaction } from '@app/types';

@Component({
  selector: 'app-transaction-detail',
  templateUrl: './transaction-detail.page.html',
  imports: [CommonModule, RouterModule],
})
export class TransactionDetailPage implements OnInit, OnDestroy {
  private store = inject(Store);
  private route = inject(ActivatedRoute);
  private flowbite = inject(FlowbiteService);

  @ViewChild('categoryDropdown') categoryDropdown!: ElementRef;
  @ViewChild('categoryDropdownButton') categoryDropdownButton!: ElementRef;

  categoriesDropdown: Dropdown | null = null;

  walletID = this.route.snapshot.params['walletID'];
  transactionID = this.route.snapshot.params['transactionID'];
  transactionLoaded$ = new Subject<boolean>();
  destroy$ = new Subject<void>();
  categories$ = this.store.select(getCategories);
  transaction$ = this.store.select(getSelectedTransaction).pipe(
    filter((v) => !!v),
    tap(() => {
      this.flowbite.load(() => initFlowbite());
      this.transactionLoaded$.next(true);
    }),
  );

  ngOnInit() {
    this.store.dispatch(actions.getTransaction({ walletID: this.walletID, transactionID: this.transactionID }));
    this.store.dispatch(actions.getCategories({ walletID: this.walletID }));

    this.transactionLoaded$
      .pipe(debounceTime(800), takeUntil(this.destroy$))
      .subscribe(() => this.setupCategoryDropdown());
  }

  ngOnDestroy() {
    this.store.dispatch(actions.setSelectedTransaction({ transaction: null }));
    this.destroy$.next();
    this.destroy$.complete();
    this.transactionLoaded$.complete();
  }

  setupCategoryDropdown() {
    this.categoriesDropdown = new Dropdown(
      this.categoryDropdown.nativeElement,
      this.categoryDropdownButton.nativeElement,
    );
  }

  updateCategory(transaction: Transaction, categoryID: string) {
    this.store.dispatch(
      actions.updateTransaction({
        walletID: this.walletID,
        transactionID: this.transactionID,
        transaction: { ...transaction, categoryID },
      }),
    );

    this.categoriesDropdown?.hide();
    this.categoryDropdownButton.nativeElement.focus();
  }
}
