import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { MyPermissions } from '../domain/rbac.model';

@Injectable({ providedIn: 'root' })
export class RbacHttpRepository {
  private readonly http = inject(HttpClient);
  private readonly apiUrl = environment.apiUrl;

  getMyPermissions(): Observable<MyPermissions> {
    return this.http.get<MyPermissions>(`${this.apiUrl}/api/v1/rbac/me/permissions`);
  }
}
