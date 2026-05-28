import { HttpErrorResponse } from '@angular/common/http';
import { extractSyncActionError } from './sync-error-parser';

describe('sync-error-parser', () => {
  it('extracts sync action error with reauth and retry metadata', () => {
    const result = extractSyncActionError({
      original: new HttpErrorResponse({ status: 429 }),
      jsonApiErrors: [
        {
          code: 'ERR_SYNC_RATE_LIMITED',
          title: 'Too Many Requests',
          detail: 'La cuenta de Outlook requiere espera temporal.',
          links: { about: 'https://help.bowerbird.dev/errors/ERR_SYNC_RATE_LIMITED' },
          meta: {
            provider: 'outlook',
            requires_reauth: false,
            retry_after_seconds: 120,
          },
        },
      ],
    });

    expect(result).toEqual({
      code: 'ERR_SYNC_RATE_LIMITED',
      traceId: undefined,
      title: 'Too Many Requests',
      message: 'La cuenta de Outlook requiere espera temporal.',
      provider: 'OUTLOOK',
      requiresReauth: false,
      retryAfterSeconds: 120,
      helpUrl: 'https://help.bowerbird.dev/errors/ERR_SYNC_RATE_LIMITED',
    });
  });

  it('returns null for non sync errors', () => {
    const result = extractSyncActionError({
      original: new HttpErrorResponse({ status: 400 }),
      jsonApiErrors: [{ code: 'ERR_VALIDATION', detail: 'invalid' }],
    });

    expect(result).toBeNull();
  });
});
