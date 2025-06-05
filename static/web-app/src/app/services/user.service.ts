import { map } from 'rxjs/operators';

import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import { environment } from '../../environments/environment';

@Injectable({
  providedIn: 'root',
})
export class UserService {
  private httpClient = inject(HttpClient);
  private apiUrl = environment.apiBaseUrl;

  setFilePasswords(passwords: string[]) {
    return this.httpClient
      .put<{ data: string }>(`${this.apiUrl}/v1/file-passwords`, { passwords })
      .pipe(map((r) => r.data));
  }
}
