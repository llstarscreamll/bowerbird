import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { AuthTokens, TenantMembership } from '../domain/auth.model';
import { environment } from '../../../environments/environment';

@Injectable({ providedIn: 'root' })
export class AuthHttpService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/api/v1/auth`;
  private readonly identityUrl = `${environment.apiUrl}/api/v1/identity`;

  loginLocal(email: string, password: string): Observable<AuthTokens> {
    return this.http.post<AuthTokens>(`${this.baseUrl}/login-local`, { email, password });
  }

  registerLocal(email: string, password: string): Observable<AuthTokens> {
    return this.http.post<AuthTokens>(`${this.baseUrl}/register-local`, { email, password });
  }

  refreshToken(): Observable<AuthTokens> {
    return this.http.post<AuthTokens>(`${this.baseUrl}/refresh`, {});
  }

  logout(): Observable<void> {
    return this.http.post<void>(`${this.baseUrl}/logout`, {});
  }

  getUserTenants(): Observable<TenantMembership[]> {
    return this.http.get<TenantMembership[]>(`${this.identityUrl}/tenants`);
  }
}
