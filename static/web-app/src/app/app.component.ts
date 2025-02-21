import { Store } from '@ngrx/store';
import { RouterOutlet } from '@angular/router';
import { Component, inject, OnInit } from '@angular/core';

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
