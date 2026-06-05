import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, switchMap, map } from 'rxjs';
import { FileReference, PresignedDownloadResponse, PresignedUploadRequest, PresignedUploadResponse } from '../domain/file-storage.model';
import { environment } from '../../../environments/environment';

export type FileUploadEvent =
  | {
      type: 'progress';
      progress: number;
    }
  | {
      type: 'completed';
      reference: FileReference;
    };

@Injectable({ providedIn: 'root' })
export class FileStorageService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/api/v1/files`;

  uploadFile(file: File, moduleName: string): Observable<FileUploadEvent> {
    return this.requestUploadUrl({
      filename: file.name,
      content_type: file.type || 'application/octet-stream',
      module: moduleName,
    }).pipe(switchMap((presignedUpload) => this.uploadToPresignedUrl(file, presignedUpload)));
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

  private uploadToPresignedUrl(file: File, presignedUpload: PresignedUploadResponse): Observable<FileUploadEvent> {
    return new Observable<FileUploadEvent>((observer) => {
      const xhr = new XMLHttpRequest();
      xhr.open(presignedUpload.method, presignedUpload.url, true);

      for (const [header, value] of Object.entries({
        ...presignedUpload.headers,
        'Content-Type': file.type || 'application/octet-stream',
      })) {
        xhr.setRequestHeader(header, value);
      }

      xhr.upload.onprogress = (event) => {
        if (!event.lengthComputable || event.total === 0) {
          return;
        }

        const progress = Math.min(99, Math.round((event.loaded / event.total) * 100));
        observer.next({ type: 'progress', progress });
      };

      xhr.onerror = () => {
        observer.error(new Error('failed to upload file to storage: network error'));
      };

      xhr.onload = () => {
        if (xhr.status < 200 || xhr.status >= 300) {
          observer.error(new Error(`failed to upload file to storage: ${xhr.status} ${xhr.responseText || ''}`));
          return;
        }

        observer.next({ type: 'progress', progress: 100 });
        observer.next({ type: 'completed', reference: presignedUpload.reference });
        observer.complete();
      };

      xhr.send(file);

      return () => {
        if (xhr.readyState !== XMLHttpRequest.DONE) {
          xhr.abort();
        }
      };
    });
  }
}
