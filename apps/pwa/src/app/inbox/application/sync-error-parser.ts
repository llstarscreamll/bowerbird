import { EnrichedHttpError, JSONAPIError, isEnrichedHttpError } from '../../core/http/jsonapi-error';

export interface SyncActionError {
  code: string;
  traceId?: string;
  title: string;
  message: string;
  provider?: string;
  requiresReauth: boolean;
  retryAfterSeconds: number;
  helpUrl?: string;
}

export function extractSyncActionError(error: unknown): SyncActionError | null {
  if (!isEnrichedHttpError(error)) {
    return null;
  }

  const firstError = firstSyncError(error);
  if (!firstError) {
    return null;
  }

  const retryAfterSeconds = normalizeRetryAfter(firstError.meta?.retry_after_seconds);

  return {
    code: firstError.code || 'ERR_SYNC_INTERNAL',
    traceId: firstError.id,
    title: firstError.title || 'Error de sincronización',
    message: firstError.detail || 'No se pudo completar la sincronización de correo.',
    provider: normalizeProvider(firstError.meta?.provider),
    requiresReauth: firstError.meta?.requires_reauth === true,
    retryAfterSeconds,
    helpUrl: firstError.links?.about,
  };
}

function firstSyncError(error: EnrichedHttpError): JSONAPIError | undefined {
  return error.jsonApiErrors.find((candidate) => (candidate.code || '').startsWith('ERR_SYNC_'));
}

function normalizeRetryAfter(value: unknown): number {
  if (typeof value !== 'number') {
    return 0;
  }

  if (!Number.isFinite(value) || value <= 0) {
    return 0;
  }

  return Math.floor(value);
}

function normalizeProvider(value: unknown): string | undefined {
  if (typeof value !== 'string') {
    return undefined;
  }

  const normalized = value.trim().toUpperCase();
  return normalized || undefined;
}
