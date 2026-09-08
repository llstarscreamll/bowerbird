export type E2EOrigins = {
  app: string;
  api: string;
  media: string;
};

const LOCAL: E2EOrigins = {
  app: 'https://app.bowerbird.dev',
  api: 'https://api.bowerbird.dev',
  media: 'https://media.bowerbird.dev',
};

const env = (name: string): string | undefined => {
  const value = process.env[name]?.trim();
  return value ? value : undefined;
};

const stripTrailingSlash = (value: string): string => value.replace(/\/+$/, '');

const parseOrigin = (value: string, name: string): string => {
  let url: URL;
  try {
    url = new URL(value);
  } catch {
    throw new Error(`${name} is not a valid URL: ${value}`);
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    throw new Error(`${name} must be http(s): ${value}`);
  }
  return url.origin;
};

function siblingOrigin(appOrigin: string, subdomain: 'api' | 'media'): string | undefined {
  const url = new URL(appOrigin);
  const labels = url.hostname.split('.');
  if (labels[0] !== 'app' && labels.length < 3) {
    return undefined;
  }
  labels[0] = subdomain;
  url.hostname = labels.join('.');
  return url.origin;
}

export function resolveE2EOrigins(): E2EOrigins {
  const appOverride = env('E2E_BASE_URL');
  const app = parseOrigin(stripTrailingSlash(appOverride ?? LOCAL.app), 'E2E_BASE_URL');

  const apiRaw = env('E2E_API_BASE_URL') ?? siblingOrigin(app, 'api') ?? (appOverride ? undefined : LOCAL.api);
  const mediaRaw = env('E2E_MEDIA_BASE_URL') ?? siblingOrigin(app, 'media') ?? (appOverride ? undefined : LOCAL.media);

  if (!apiRaw) {
    throw new Error(`Cannot derive API origin from ${app}. Set E2E_API_BASE_URL.`);
  }
  if (!mediaRaw) {
    throw new Error(`Cannot derive media origin from ${app}. Set E2E_MEDIA_BASE_URL.`);
  }

  return {
    app,
    api: parseOrigin(stripTrailingSlash(apiRaw), 'E2E_API_BASE_URL'),
    media: parseOrigin(stripTrailingSlash(mediaRaw), 'E2E_MEDIA_BASE_URL'),
  };
}

export function appOrigin(): string {
  return resolveE2EOrigins().app;
}

export function apiOrigin(): string {
  return resolveE2EOrigins().api;
}

export function mediaOrigin(): string {
  return resolveE2EOrigins().media;
}
