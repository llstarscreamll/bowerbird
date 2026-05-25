import { HttpInterceptorFn } from '@angular/common/http';
import { createUlid } from '../utils/id.util';

export const requestIdInterceptor: HttpInterceptorFn = (req, next) => {
  if (req.headers.has('X-Request-ID')) {
    return next(req);
  }

  const clonedReq = req.clone({
    headers: req.headers.set('X-Request-ID', createUlid()),
  });

  return next(clonedReq);
};
