import DOMPurify from 'dompurify';

export interface SecuredEmailHtml {
  sanitizedHtml: string;
  blockedExternalImages: number;
}

export function secureEmailHtml(htmlContent: string, allowExternalImages: boolean): SecuredEmailHtml {
  const source = (htmlContent || '').trim();
  if (!source) {
    return {
      sanitizedHtml: '',
      blockedExternalImages: 0,
    };
  }

  const sanitized = DOMPurify.sanitize(source, {
    USE_PROFILES: { html: true },
    FORBID_TAGS: ['script', 'style', 'iframe', 'object', 'embed', 'link', 'meta', 'base', 'form'],
    FORBID_ATTR: ['srcset', 'onerror', 'onload', 'onclick', 'onmouseover', 'onfocus', 'style'],
  });

  const parser = new DOMParser();
  const document = parser.parseFromString(sanitized, 'text/html');

  for (const link of Array.from(document.querySelectorAll('a[href]'))) {
    link.setAttribute('target', '_blank');
    link.setAttribute('rel', 'noopener noreferrer');
  }

  let blockedExternalImages = 0;
  for (const image of Array.from(document.querySelectorAll('img'))) {
    const sourceURL = (image.getAttribute('src') || '').trim();
    if (!isExternalImage(sourceURL)) {
      continue;
    }

    if (!allowExternalImages) {
      blockedExternalImages += 1;
      image.setAttribute('data-blocked-src', sourceURL);
      image.setAttribute('src', '');
      image.setAttribute('alt', image.getAttribute('alt') || 'Imagen externa bloqueada');
    }
  }

  return {
    sanitizedHtml: document.body.innerHTML,
    blockedExternalImages,
  };
}

function isExternalImage(source: string): boolean {
  const normalized = source.trim().toLowerCase();
  if (!normalized) {
    return false;
  }

  return normalized.startsWith('http://') || normalized.startsWith('https://') || normalized.startsWith('//');
}
