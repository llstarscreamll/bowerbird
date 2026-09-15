import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { CreateLegalEntityInput, LegalEntity, UpdateLegalEntityInput } from '../domain/legal-entity.model';

type LegalEntityDoc = { id: string; attributes: Omit<LegalEntity, 'id'> };

@Injectable({ providedIn: 'root' })
export class LegalEntitiesHttpService {
  private readonly http = inject(HttpClient);
  private readonly apiDomain = environment.apiUrl;

  list(): Observable<LegalEntity[]> {
    return this.http.get<{ data: LegalEntityDoc[] }>(`${this.apiDomain}/api/v1/legal-entities`).pipe(map((res) => res.data.map((doc) => ({ id: doc.id, ...doc.attributes }))));
  }

  create(input: CreateLegalEntityInput): Observable<LegalEntity> {
    return this.http
      .post<{ data: LegalEntityDoc }>(`${this.apiDomain}/api/v1/legal-entities`, {
        data: { type: 'legal-entities', attributes: input },
      })
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }

  update(id: string, input: UpdateLegalEntityInput): Observable<LegalEntity> {
    return this.http
      .patch<{ data: LegalEntityDoc }>(`${this.apiDomain}/api/v1/legal-entities/${id}`, {
        data: { attributes: input },
      })
      .pipe(map((res) => ({ id: res.data.id, ...res.data.attributes })));
  }
}
