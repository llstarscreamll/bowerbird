import { ActionReducer, ActionReducerMap, MetaReducer } from '@ngrx/store';

import { isDevMode } from '@angular/core';

import * as auth from './auth';
import * as finance from './finance';

interface AppState {
  auth: auth.State;
  finance: finance.State;
}

export const reducers: ActionReducerMap<AppState> = {
  auth: auth.reducer,
  finance: finance.reducer,
};

export const metaReducers: MetaReducer<AppState>[] = isDevMode() ? [debug] : [];

function debug(reducer: ActionReducer<any>): ActionReducer<any> {
  return function (state, action: any) {
    const r = reducer(state, action);
    console.log(action.type + ':', {
      props: Object.keys(action).reduce((acc, key) => {
        if (key === 'type') {
          return acc;
        }
        return { ...acc, [key]: action[key] };
      }, {}),
      0: state,
      1: r,
    });

    return r;
  };
}
