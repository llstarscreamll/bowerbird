import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { DOCUMENT } from '@angular/common';

const GLOBAL_ROUTES = new Set(['', 'login', 'lobby', 'workspaces', 'onboarding', 'profile']);

export const tenantInterceptor: HttpInterceptorFn = (req, next) => {
  const document = inject(DOCUMENT);
  const pathname = document.defaultView?.location.pathname || '';

  // Example logic:
  // Path format: app.bowerbird.com/organization-slug/dashboard
  // We extract the first path segment
  let tenantId = '';
  const parts = pathname.split('/');

  // parts[0] is always '' because pathname starts with '/'
  const firstSegment = parts[1];

  if (firstSegment && !GLOBAL_ROUTES.has(firstSegment)) {
    tenantId = firstSegment;
  }

  // If an organization slug was found, append the tenant header
  if (tenantId) {
    const clonedReq = req.clone({
      headers: req.headers.set('X-Tenant-ID', tenantId),
    });
    return next(clonedReq);
  }

  return next(req);
};
