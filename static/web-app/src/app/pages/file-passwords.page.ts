import { Store } from '@ngrx/store';

import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { RouterModule } from '@angular/router';

import { actions } from '@app/ngrx/finance';

@Component({
  imports: [CommonModule, RouterModule, ReactiveFormsModule],
  selector: 'app-file-passwords-page',
  templateUrl: './file-passwords.page.html',
})
export class FilePasswordsPageComponent {
  private store = inject(Store);

  form = new FormGroup({
    passwords: new FormControl('', [Validators.required, Validators.minLength(1)]),
  });

  onSubmit() {
    if (this.form.invalid) {
      return;
    }

    this.store.dispatch(
      actions.setFilePasswords({
        passwords: this.form.value.passwords?.split('\n').filter((p) => p.trim() !== '') ?? [],
      }),
    );
  }
}
