export type FileUploadQueueItemStatus = 'pending' | 'uploading' | 'uploaded' | 'failed';

export interface FileUploadQueueItem {
  id: string;
  name: string;
  size: number;
  status: FileUploadQueueItemStatus;
  progress: number;
}
