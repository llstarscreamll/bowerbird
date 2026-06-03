export interface PresignedUploadRequest {
  filename: string;
  content_type: string;
  module: string;
}

export interface FileReference {
  bucket: string;
  key: string;
}

export interface PresignedUploadResponse {
  url: string;
  method: 'PUT';
  headers: Record<string, string>;
  expires_at: string;
  reference: FileReference;
  upload_path: string;
}

export interface PresignedDownloadRequest {
  key: string;
}

export interface PresignedDownloadResponse {
  url: string;
  method: 'GET';
  expires_at: string;
  reference: FileReference;
}
