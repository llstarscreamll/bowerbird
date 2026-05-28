import { HttpErrorResponse } from '@angular/common/http';

export interface JSONAPIErrorMeta {
  _debug?: unknown;
  requires_reauth?: boolean;
  provider?: string;
  retry_after_seconds?: number;
  account_email?: string;
  [key: string]: unknown;
}

export interface JSONAPIErrorLinks {
  about?: string;
}

export interface JSONAPIError {
  id?: string;
  status?: string;
  code?: string;
  title?: string;
  detail?: string;
  links?: JSONAPIErrorLinks;
  meta?: JSONAPIErrorMeta;
}

export interface EnrichedHttpError {
  original: HttpErrorResponse;
  jsonApiErrors: JSONAPIError[];
}

export function isEnrichedHttpError(error: unknown): error is EnrichedHttpError {
  if (!error || typeof error !== 'object') {
    return false;
  }

  const candidate = error as Partial<EnrichedHttpError>;
  return Array.isArray(candidate.jsonApiErrors) && candidate.original instanceof HttpErrorResponse;
}
