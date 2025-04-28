import { of } from 'rxjs';
import { catchError, map, switchMap } from 'rxjs/operators';

import { Actions, createEffect, ofType } from '@ngrx/effects';
import { createActionGroup, createReducer, emptyProps, on, props } from '@ngrx/store';
import { createFeatureSelector, createSelector } from '@ngrx/store';

import { HttpErrorResponse } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import * as auth from '@app/ngrx/auth';
import { WalletService } from '@app/services/wallet.service';
import { Category, Transaction, Wallet } from '@app/types';

export enum Status {
  empty = '',
  loading = 'loading',
  ok = 'ok',
  error = 'error',
}

export interface State {
  status: Status;
  wallets: Wallet[];
  selectedWallet: Wallet | null;
  transactions: Transaction[];
  error: HttpErrorResponse | null;
  transaction: Transaction | null;
  categories: Category[];
}

export const initialState: State = {
  status: Status.empty,
  wallets: [],
  selectedWallet: null,
  transactions: [],
  error: null,
  transaction: null,
  categories: [],
};

export const actions = createActionGroup({
  source: 'Finance',
  events: {
    'get wallets': emptyProps(),
    'get wallets ok': props<{ wallets: Wallet[] }>(),
    'get wallets error': props<{ error: HttpErrorResponse }>(),

    'set selected wallet': props<{ wallet: Wallet }>(),

    'sync transactions from email': props<{ walletID: string }>(),

    'get transactions': props<{ walletID: string }>(),
    'get transactions ok': props<{ transactions: any[] }>(),
    'get transactions error': props<{ error: HttpErrorResponse }>(),

    'get transaction': props<{ walletID: string; transactionID: string }>(),
    'get transaction ok': props<{ transaction: Transaction }>(),
    'get transaction error': props<{ error: HttpErrorResponse }>(),
    'set selected transaction': props<{ transaction: Transaction | null }>(),

    'get categories': props<{ walletID: string }>(),
    'get categories ok': props<{ categories: Category[] }>(),
    'get categories error': props<{ error: HttpErrorResponse }>(),
  },
});

export const reducer = createReducer(
  initialState,
  on(actions.getWallets, (state) => ({ ...state, status: Status.loading })),
  on(actions.getWalletsOk, (state, { wallets }) => ({ ...state, status: Status.ok, wallets })),
  on(actions.getWalletsError, (state, { error }) => ({ ...state, status: Status.error, error })),
  on(actions.setSelectedWallet, (state, { wallet }) => ({ ...state, selectedWallet: wallet })),

  on(actions.getTransactions, (state) => ({ ...state, status: Status.loading })),
  on(actions.getTransactionsOk, (state, { transactions }) => ({ ...state, transactions, status: Status.ok })),

  on(actions.getTransaction, (state) => ({ ...state, status: Status.loading })),
  on(actions.getTransactionOk, (state, { transaction }) => ({ ...state, transaction, status: Status.ok })),
  on(actions.getTransactionError, (state, { error }) => ({ ...state, error, status: Status.error })),
  on(actions.setSelectedTransaction, (state, { transaction }) => ({ ...state, transaction })),

  on(actions.getCategories, (state) => ({ ...state, status: Status.loading })),
  on(actions.getCategoriesOk, (state, { categories }) => ({ ...state, categories, status: Status.ok })),
  on(actions.getCategoriesError, (state, { error }) => ({ ...state, error, status: Status.error })),
);

export const getFinanceState = createFeatureSelector<State>('finance');
export const getSelectedWallet = createSelector(getFinanceState, (state: State) => state.selectedWallet);
export const getTransactions = createSelector(getFinanceState, (state: State) => state.transactions);
export const getTransaction = createSelector(getFinanceState, (state: State) => state.transaction);
export const getCategories = createSelector(getFinanceState, (state: State) => state.categories);

@Injectable()
export class Effects {
  private actions$ = inject(Actions);
  private walletService = inject(WalletService);

  getWalletsAfterGettingUserSucceed$ = createEffect(() =>
    this.actions$.pipe(
      ofType(auth.actions.getUserOk),
      map(() => actions.getWallets()),
    ),
  );

  getWallets$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.getWallets),
      switchMap(() =>
        this.walletService.search().pipe(
          map((wallets) => actions.getWalletsOk({ wallets })),
          catchError((error) => of(actions.getWalletsError({ error }))),
        ),
      ),
    ),
  );

  setSelectedWalletWhenWalletsLoaded$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.getWalletsOk),
      map(({ wallets }) => actions.setSelectedWallet({ wallet: wallets[0] })),
    ),
  );

  getWalletTransactions$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.setSelectedWallet),
      map(({ wallet }) => actions.getTransactions({ walletID: wallet.ID })),
    ),
  );

  getTransactions$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.getTransactions),
      switchMap(({ walletID }) =>
        this.walletService.searchTransactions(walletID).pipe(
          map((transactions) => actions.getTransactionsOk({ transactions })),
          catchError((error) => of(actions.getTransactionsError({ error }))),
        ),
      ),
    ),
  );

  syncTransactionsFromEmail$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.syncTransactionsFromEmail),
      switchMap(({ walletID }) =>
        this.walletService.syncTransactionsFromEmail(walletID).pipe(
          map(() => actions.getTransactions({ walletID })),
          catchError((error) => of(actions.getTransactionsError({ error }))),
        ),
      ),
    ),
  );

  getTransaction$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.getTransaction),
      switchMap(({ walletID, transactionID }) =>
        this.walletService.getTransaction(walletID, transactionID).pipe(
          map((transaction) => actions.getTransactionOk({ transaction })),
          catchError((error) => of(actions.getTransactionError({ error }))),
        ),
      ),
    ),
  );

  getCategories$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.getCategories),
      switchMap(({ walletID }) =>
        this.walletService.getCategories(walletID).pipe(
          map((categories) => actions.getCategoriesOk({ categories })),
          catchError((error) => of(actions.getCategoriesError({ error }))),
        ),
      ),
    ),
  );
}
