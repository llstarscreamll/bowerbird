import { Store } from '@ngrx/store';

import { Component, OnInit, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';

import * as auth from './ngrx/auth';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  template: '<router-outlet></router-outlet>',
})
export class AppComponent implements OnInit {
  store = inject(Store);

  ngOnInit(): void {
    this.store.dispatch(auth.actions.getUser());
  }
}
