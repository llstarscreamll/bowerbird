export const CATALOG_IMPORT_MAX_FILE_BYTES = 500 * 1024 * 1024;
export const CATALOG_IMPORT_ACCEPT = '.csv,text/csv,text/plain';
export const CATALOG_IMPORT_UPLOAD_MODULE = 'catalog';

export interface CatalogImportActor {
  user_id: string;
  email: string;
  name: string;
}

export interface CatalogImport {
  id: string;
  file_key: string;
  file_size_bytes: number;
  status: string;
  total_rows: number;
  created_count: number;
  updated_count: number;
  failed_count: number;
  processed: number;
  byte_offset: number;
  failure_reason?: string;
  requested_by: CatalogImportActor;
  cancelled_by: CatalogImportActor | null;
  created_at: string;
  updated_at: string;
  completed_at: string | null;
  cancelled_at: string | null;
}

export interface CatalogImportError {
  id: string;
  file_row: number;
  column?: string;
  internal_code?: string;
  name?: string;
  kind?: string;
  code: string;
  message: string;
}

export interface CatalogPage<T> {
  items: T[];
  hasMore: boolean;
  cursor?: string;
  total?: number;
}

export function catalogImportStatusLabel(status: string): string {
  switch (status) {
    case 'queued':
      return 'En cola';
    case 'processing':
      return 'Procesando';
    case 'completed':
      return 'Completado';
    case 'failed':
      return 'Fallido';
    case 'cancelled':
      return 'Cancelado';
    default:
      return status;
  }
}

export function catalogImportIsActive(status: string): boolean {
  return status === 'queued' || status === 'processing';
}

export function catalogImportColumnLabel(column?: string): string {
  switch (column) {
    case 'internal_code':
      return 'Código interno';
    case 'name':
      return 'Nombre';
    case 'kind':
      return 'Tipo';
    default:
      return 'Fila';
  }
}

export function catalogImportProgressPercent(imp: CatalogImport | null): number {
  if (!imp) return 0;
  if (imp.total_rows > 0) {
    return Math.min(100, Math.round((imp.processed / imp.total_rows) * 100));
  }
  if (imp.file_size_bytes > 0) {
    return Math.min(99, Math.round((imp.byte_offset / imp.file_size_bytes) * 100));
  }
  return catalogImportIsActive(imp.status) ? 0 : 100;
}
