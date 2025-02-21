import { of } from 'rxjs';
import { catchError, map, switchMap } from 'rxjs/operators';

import { Actions, createEffect, ofType } from '@ngrx/effects';
import {
  createActionGroup,
  createFeatureSelector,
  createReducer,
  createSelector,
  emptyProps,
  on,
  props,
} from '@ngrx/store';

import { HttpErrorResponse } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import { AuthService } from '@app/services/auth.service';
import { User } from '@app/types';

enum Status {
  empty = '',
  loading = 'loading',
  loggedIn = 'authenticated',
  notLoggedIn = 'unauthenticated',
  error = 'error',
}

export interface State {
  status: Status;
  error: HttpErrorResponse | null;
  user: User | null;
}

export const initialState: State = {
  status: Status.empty,
  error: null,
  user: null,
};

export const actions = createActionGroup({
  source: 'Auth',
  events: {
    'get user': emptyProps(),
    'get user ok': props<{ user: User }>(),
    'get user error': props<{ error: HttpErrorResponse }>(),
  },
});

export const reducer = createReducer(
  initialState,
  on(actions.getUser, (s) => {
    return { ...s, status: Status.loading };
  }),
  on(actions.getUserOk, (s, { user }) => ({
    ...s,
    user,
    error: null,
    status: Status.loggedIn,
  })),
  on(actions.getUserError, (s, { error }) => ({
    ...s,
    user: null,
    error: error,
    status: error.status === 401 ? Status.notLoggedIn : Status.error,
  })),
);

const getAuthState = createFeatureSelector<State>('auth');
export const getLoggedIn = createSelector(getAuthState, (s) => s.status === Status.loggedIn);
export const getUser = createSelector(getAuthState, (s) => s.user);

@Injectable()
export class Effects {
  private actions$ = inject(Actions);
  private authService = inject(AuthService);

  getUser$ = createEffect(() =>
    this.actions$.pipe(
      ofType(actions.getUser),
      switchMap(() =>
        this.authService.getAuthUser().pipe(
          map((user) => actions.getUserOk({ user })),
          catchError((error) => of(actions.getUserError({ error }))),
        ),
      ),
    ),
  );
}
