import { map } from 'rxjs/operators';

import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import { User, Wallet } from '@app/types';
import { environment } from '@env/environment';

@Injectable({
  providedIn: 'root',
})
export class WalletService {
  httpClient = inject(HttpClient);
  apiUrl = environment.apiBaseUrl;

  search() {
    return this.httpClient.get<{ data: Wallet[] }>(`${this.apiUrl}/v1/wallets`).pipe(map((r) => r.data));
  }
}
