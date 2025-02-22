import { of } from 'rxjs';
import { catchError, map, switchMap } from 'rxjs/operators';

import { Actions, createEffect, ofType } from '@ngrx/effects';
import { createAction, createActionGroup, createReducer, emptyProps, on, props } from '@ngrx/store';
import { createFeatureSelector, createSelector } from '@ngrx/store';

import { HttpErrorResponse } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import * as auth from '@app/ngrx/auth';
import { WalletService } from '@app/services/wallet.service';
import { Wallet } from '@app/types';

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
  transactions: Array<{ id: number; amount: number; description: string }>;
  error: HttpErrorResponse | null;
}

export const initialState: State = {
  status: Status.empty,
  wallets: [],
  selectedWallet: null,
  transactions: [],
  error: null,
};

export const actions = createActionGroup({
  source: 'Finance',
  events: {
    'get wallets': emptyProps(),
    'get wallets ok': props<{ wallets: Wallet[] }>(),
    'get wallets error': props<{ error: HttpErrorResponse }>(),

    'set selected wallet': props<{ wallet: Wallet }>(),

    'get transactions': props<{ walletID: string }>(),
    'get transactions ok': props<{ transactions: any[] }>(),
    'get transactions error': props<{ error: HttpErrorResponse }>(),
  },
});

export const reducer = createReducer(
  initialState,
  on(actions.getWallets, (state) => ({ ...state, status: Status.loading })),
  on(actions.getWalletsOk, (state, { wallets }) => ({ ...state, status: Status.ok, wallets })),
  on(actions.getWalletsError, (state, { error }) => ({ ...state, status: Status.error, error })),
  on(actions.setSelectedWallet, (state, { wallet }) => ({ ...state, selectedWallet: wallet })),
);

export const getFinanceState = createFeatureSelector<State>('finance');
export const getSelectedWallet = createSelector(getFinanceState, (state: State) => state.selectedWallet);
export const getTransactions = createSelector(getFinanceState, (state: State) => state.transactions);

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
}
