import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, from, switchMap, map } from 'rxjs';
import { FileReference, PresignedDownloadResponse, PresignedUploadRequest, PresignedUploadResponse } from '../domain/file-storage.model';
import { environment } from '../../../environments/environment';

@Injectable({ providedIn: 'root' })
export class FileStorageService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/api/v1/files`;

  uploadFile(file: File, moduleName: string): Observable<FileReference> {
    return this.requestUploadUrl({
      filename: file.name,
      content_type: file.type || 'application/octet-stream',
      module: moduleName,
    }).pipe(
      switchMap((presignedUpload) =>
        from(
          fetch(presignedUpload.url, {
            method: presignedUpload.method,
            headers: {
              ...presignedUpload.headers,
              'Content-Type': file.type || 'application/octet-stream',
            },
            body: file,
          }),
        ).pipe(
          switchMap(async (response) => {
            if (!response.ok) {
              const body = await response.text();
              throw new Error(`failed to upload file to storage: ${response.status} ${body}`);
            }

            return presignedUpload.reference;
          }),
        ),
      ),
    );
  }

  requestDownloadUrl(key: string): Observable<string> {
    return this.http
      .post<PresignedDownloadResponse>(`${this.baseUrl}/downloads/presigned`, {
        key,
      })
      .pipe(map((response) => response.url));
  }

  private requestUploadUrl(input: PresignedUploadRequest): Observable<PresignedUploadResponse> {
    return this.http.post<PresignedUploadResponse>(`${this.baseUrl}/uploads/presigned`, input);
  }
}
