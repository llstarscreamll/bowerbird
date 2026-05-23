import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, catchError, map, of } from 'rxjs';
import { HealthRepository } from '../domain/health.repository';
import { HealthInfo } from '../domain/health.model';
import { environment } from '../../../environments/environment';

interface HealthDto {
  status: string;
}

@Injectable({
  providedIn: 'root',
})
export class HealthHttpService implements HealthRepository {
  private readonly http = inject(HttpClient);

  checkHealth(): Observable<HealthInfo> {
    return this.http.get<HealthDto>(`${environment.apiUrl}/api/health`).pipe(
      map((res) => ({
        status: res.status as 'ok' | 'degraded',
        lastChecked: new Date(),
      })),
      catchError(() =>
        of({
          status: 'degraded' as const,
          lastChecked: new Date(),
        }),
      ),
    );
  }
}
