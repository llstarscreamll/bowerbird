import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';
import { AuthTokens, TenantMembership } from './auth.model';

export interface AuthRepository {
  loginLocal(email: string, password: string): Observable<AuthTokens>;
  registerLocal(email: string, password: string): Observable<AuthTokens>;
  refreshToken(): Observable<AuthTokens>;
  logout(): Observable<void>;
  getUserTenants(): Observable<TenantMembership[]>;
}

export const AUTH_REPOSITORY = new InjectionToken<AuthRepository>('AUTH_REPOSITORY');
