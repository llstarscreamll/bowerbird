import { map } from 'rxjs/operators';

import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import { User } from '@app/types';
import { environment } from '@env/environment';

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  httpClient = inject(HttpClient);
  apiUrl = environment.apiBaseUrl;

  getAuthUser() {
    return this.httpClient.get<{ data: User }>(`${this.apiUrl}/v1/auth/user`).pipe(map((r) => r.data));
  }
}
