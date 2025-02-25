import { map } from 'rxjs/operators';

import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import { Transaction, User, Wallet } from '@app/types';
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

  searchTransactions(walletID: string) {
    return this.httpClient
      .get<{ data: Transaction[] }>(`${this.apiUrl}/v1/wallets/${walletID}/transactions`)
      .pipe(map((r) => r.data));
  }

  syncTransactionsFromEmail(walletID: string) {
    return this.httpClient
      .post<{ data: string }>(`${this.apiUrl}/v1/wallets/${walletID}/transactions/sync-from-mail`, {})
      .pipe(map((r) => r.data));
  }
}
