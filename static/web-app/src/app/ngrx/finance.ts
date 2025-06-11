import { of } from 'rxjs';
import { catchError, map, mergeMap, switchMap, tap } from 'rxjs/operators';

import { Actions, createEffect, ofType } from '@ngrx/effects';
import { createActionGroup, createReducer, emptyProps, on, props } from '@ngrx/store';
import { createFeatureSelector, createSelector } from '@ngrx/store';

import { HttpErrorResponse } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Router } from '@angular/router';

import * as auth from '@app/ngrx/auth';
import { UserService } from '@app/services/user.service';
import { WalletService } from '@app/services/wallet.service';
import { Category, Transaction, Wallet } from '@app/types';

export enum Status {
  empty = '',
  loading = 'loading',
  ok = 'ok',
  error = 'error',
}

export type WalletMetrics = {
  walletID: string;
  from: string;
  to: string;
  totalIncome: number;
  totalExpense: number;
  expensesByCategory: {
    categoryName: string;
    total: number;
    color: string;
  }[];
};

export interface State {
  status: Status;
  wallets: Wallet[];
  selectedWallet: Wallet | null;
  transactions: Transaction[];
  metrics: WalletMetrics | null;
  error: HttpErrorResponse | null;
  transaction: Transaction | null;
  categories: Category[];
}

export const initialState: State = {
  status: Status.empty,
  wallets: [],
  selectedWallet: null,
  transactions: [],
  metrics: null,
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
    'sync transactions from email ok': emptyProps(),
    'sync transactions from email error': props<{ error: HttpErrorResponse }>(),

    'get metrics': props<{ walletID: string; from: Date; to: Date }>(),
    'get metrics ok': props<{ metrics: any }>(),
    'get metrics error': props<{ error: HttpErrorResponse }>(),

    'get transactions': props<{ walletID: string }>(),
    'get transactions ok': props<{ transactions: Transaction[] }>(),
    'get transactions error': props<{ error: HttpErrorResponse }>(),

    'get transaction': props<{ walletID: string; transactionID: string }>(),
    'get transaction ok': props<{ transaction: Transaction }>(),
    'get transaction error': props<{ error: HttpErrorResponse }>(),
    'set selected transaction': props<{ transaction: Transaction | null }>(),

    'get categories': props<{ walletID: string }>(),
    'get categories ok': props<{ categories: Category[] }>(),
    'get categories error': props<{ error: HttpErrorResponse }>(),

    'create category': props<{ walletID: string; category: Category }>(),
    'create category ok': props<{ response: string }>(),
    'create category error': props<{ error: HttpErrorResponse }>(),

    'update transaction': props<{ walletID: string; transactionID: string; transaction: Transaction }>(),
    'update transaction ok': props<{ walletID: string; transactionID: string }>(),
    'update transaction error': props<{ error: HttpErrorResponse }>(),

    'set file passwords': props<{ passwords: string[] }>(),
    'set file passwords ok': emptyProps(),
    'set file passwords error': props<{ error: HttpErrorResponse }>(),
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
  on(actions.getTransactionsError, (state, { error }) => ({ ...state, error, status: Status.error })),

  on(actions.syncTransactionsFromEmail, (state) => ({ ...state, status: Status.loading })),
  on(actions.syncTransactionsFromEmailOk, (state) => ({ ...state, status: Status.ok })),
  on(actions.syncTransactionsFromEmailError, (state, { error }) => ({ ...state, error, status: Status.error })),

  on(actions.getMetrics, (state) => ({ ...state, status: Status.loading })),
  on(actions.getMetricsOk, (state, { metrics }) => ({ ...state, metrics, status: Status.ok })),
  on(actions.getMetricsError, (state, { error }) => ({ ...state, error, status: Status.error })),

  on(actions.getTransaction, (state) => ({ ...state, status: Status.loading })),
  on(actions.getTransactionOk, (state, { transaction }) => ({ ...state, transaction, status: Status.ok })),
  on(actions.getTransactionError, (state, { error }) => ({ ...state, error, status: Status.error })),
  on(actions.setSelectedTransaction, (state, { transaction }) => ({ ...state, transaction })),

  on(actions.getCategories, (state) => ({ ...state, status: Status.loading })),
  on(actions.getCategoriesOk, (state, { categories }) => ({ ...state, categories, status: Status.ok })),
  on(actions.getCategoriesError, (state, { error }) => ({ ...state, error, status: Status.error })),

  on(actions.createCategory, (state) => ({ ...state, status: Status.loading })),
  on(actions.createCategoryOk, (state) => ({ ...state, status: Status.ok })),
  on(actions.createCategoryError, (state, { error }) => ({ ...state, error, status: Status.error })),

  on(actions.updateTransaction, (state) => ({ ...state, status: Status.loading })),
  on(actions.updateTransactionOk, (state) => ({ ...state, status: Status.ok })),
  on(actions.updateTransactionError, (state, { error }) => ({ ...state, error, status: Status.error })),
);

export const getFinanceState = createFeatureSelector<State>('finance');
export const getSelectedWallet = createSelector(getFinanceState, (state: State) => state.selectedWallet);
export const getTransactions = createSelector(getFinanceState, (state: State) => state.transactions);
export const getSelectedTransaction = createSelector(getFinanceState, (state: State) => state.transaction);
export const getCategories = createSelector(getFinanceState, (state: State) => state.categories);
export const getMetrics = createSelector(getFinanceState, (state: State) => state.metrics);
export const getIsLoading = createSelector(getFinanceState, (state: State) => state.status === Status.loading);

@Injectable()
export class Effects {
  private router = inject(Router);
  private actions$ = inject(Actions);
  private userService = inject(UserService);
  private walletService = inject(WalletService);

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
          mergeMap(() => [actions.syncTransactionsFromEmailOk(), actions.getTransactions({ walletID })]),
          catchError((error) => of(actions.syncTransactionsFromEmailError({ error }))),
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

  createCategory$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.createCategory),
      switchMap(({ walletID, category }) =>
        this.walletService.createCategory(walletID, category).pipe(
          map((response) => actions.createCategoryOk({ response })),
          catchError((error) => of(actions.createCategoryError({ error }))),
        ),
      ),
    ),
  );

  updateTransaction$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.updateTransaction),
      switchMap(({ walletID, transactionID, transaction }) =>
        this.walletService.updateTransaction(walletID, transactionID, transaction).pipe(
          map(() => actions.updateTransactionOk({ walletID, transactionID })),
          catchError((error) => of(actions.updateTransactionError({ error }))),
        ),
      ),
    ),
  );

  updateTransactionSuccess$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.updateTransactionOk),
      map(({ walletID, transactionID }) => actions.getTransaction({ walletID, transactionID })),
    ),
  );

  getMetrics$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.getMetrics),
      switchMap(({ walletID, from, to }) =>
        this.walletService.getMetrics(walletID, from, to).pipe(
          map((metrics) => actions.getMetricsOk({ metrics })),
          catchError((error) => of(actions.getMetricsError({ error }))),
        ),
      ),
    ),
  );

  setFilePasswords$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.setFilePasswords),
      switchMap(({ passwords }) =>
        this.userService.setFilePasswords(passwords).pipe(
          map(() => actions.setFilePasswordsOk()),
          catchError((error) => of(actions.setFilePasswordsError({ error }))),
        ),
      ),
    ),
  );

  createCategorySuccess$ = createEffect(
    () =>
      this.actions$.pipe(
        ofType(actions.createCategoryOk, actions.setFilePasswordsOk),
        tap(() => {
          this.router.navigate(['/dashboard']);
        }),
      ),
    { dispatch: false },
  );
}
