import { expect, test, type APIResponse } from '@playwright/test';

/** Weighted like https://securityheaders.com/ (Scott Helme, 160 pts). A+ requires ≥95%. */
export const SECURITY_HEADER_POINTS = {
  https: 30,
  'strict-transport-security': 25,
  'content-security-policy': 25,
  'x-frame-options': 20,
  'x-content-type-options': 20,
  'referrer-policy': 20,
  'permissions-policy': 20,
} as const;

export const SECURITY_HEADERS_MAX = 160;
const HSTS_MIN_MAX_AGE = 31536000;

const SAFE_REFERRER_POLICIES = new Set(['no-referrer', 'no-referrer-when-downgrade', 'same-origin', 'origin', 'strict-origin', 'origin-when-cross-origin', 'strict-origin-when-cross-origin']);

export type HeaderMap = Record<string, string>;

export type HeaderFinding = {
  feature: keyof typeof SECURITY_HEADER_POINTS;
  points: number;
  max: number;
  detail: string;
};

export type SecurityHeadersScan = {
  url: string;
  status: number;
  score: number;
  percent: number;
  grade: string;
  findings: HeaderFinding[];
  warnings: string[];
  headersDump: string;
};

export function normalizeHeaders(headers: HeaderMap): HeaderMap {
  const out: HeaderMap = {};
  for (const [key, value] of Object.entries(headers)) {
    out[key.toLowerCase()] = value;
  }
  return out;
}

export function headersFromResponse(response: APIResponse): HeaderMap {
  return normalizeHeaders(response.headers());
}

const header = (headers: HeaderMap, name: string): string => headers[name.toLowerCase()]?.trim() ?? '';

const parseHstsMaxAge = (value: string): number | null => {
  const match = value.match(/(?:^|;)\s*max-age=(\d+)/i);
  if (!match) {
    return null;
  }
  return Number(match[1]);
};

const cspHas = (csp: string, token: string): boolean => {
  return csp.toLowerCase().includes(token.toLowerCase());
};

const gradeForPercent = (percent: number): string => {
  if (percent >= 95) {
    return 'A+';
  }
  if (percent >= 75) {
    return 'A';
  }
  if (percent >= 60) {
    return 'B';
  }
  if (percent >= 50) {
    return 'C';
  }
  if (percent >= 29) {
    return 'D';
  }
  if (percent >= 14) {
    return 'E';
  }
  return 'F';
};

export function scanSecurityHeaders(url: string, status: number, rawHeaders: HeaderMap): SecurityHeadersScan {
  const headers = normalizeHeaders(rawHeaders);
  const warnings: string[] = [];
  const findings: HeaderFinding[] = [];

  const https = url.toLowerCase().startsWith('https://');
  findings.push({
    feature: 'https',
    points: https ? SECURITY_HEADER_POINTS.https : 0,
    max: SECURITY_HEADER_POINTS.https,
    detail: https ? url : `not https: ${url}`,
  });
  if (!https) {
    warnings.push('scan is not over HTTPS — securityheaders.com will not award https points');
  }

  const hsts = header(headers, 'strict-transport-security');
  if (!hsts) {
    findings.push({
      feature: 'strict-transport-security',
      points: 0,
      max: SECURITY_HEADER_POINTS['strict-transport-security'],
      detail: 'missing — SSL stripping / protocol downgrade is possible',
    });
  } else {
    const maxAge = parseHstsMaxAge(hsts);
    const includeSub = /(?:^|;)\s*includeSubDomains\s*(?:;|$)/i.test(hsts);
    let points: number = SECURITY_HEADER_POINTS['strict-transport-security'];
    let detail = hsts;
    if (maxAge === null) {
      points = 0;
      detail = `${hsts} (invalid: no max-age)`;
    } else if (maxAge < HSTS_MIN_MAX_AGE) {
      points = 0;
      detail = `${hsts} (max-age ${maxAge} < ${HSTS_MIN_MAX_AGE}; preload/A+ expects ≥1 year)`;
    } else if (!includeSub) {
      warnings.push(`HSTS missing includeSubDomains: ${hsts}`);
    }
    if (!/(?:^|;)\s*preload\s*(?:;|$)/i.test(hsts)) {
      warnings.push(`HSTS missing preload (recommended for A+ / hstspreload.org): ${hsts}`);
    }
    findings.push({
      feature: 'strict-transport-security',
      points,
      max: SECURITY_HEADER_POINTS['strict-transport-security'],
      detail,
    });
  }

  const cspReportOnly = header(headers, 'content-security-policy-report-only');
  const csp = header(headers, 'content-security-policy');
  if (!csp) {
    findings.push({
      feature: 'content-security-policy',
      points: 0,
      max: SECURITY_HEADER_POINTS['content-security-policy'],
      detail: cspReportOnly ? `missing enforcing CSP (only Report-Only: ${cspReportOnly})` : 'missing — XSS can load arbitrary script/style/frame',
    });
  } else {
    let points: number = SECURITY_HEADER_POINTS['content-security-policy'];
    let detail = csp;
    if (cspHas(csp, 'unsafe-eval')) {
      points = 0;
      detail = `${csp} (unsafe-eval allows arbitrary JS compilation)`;
    }
    findings.push({
      feature: 'content-security-policy',
      points,
      max: SECURITY_HEADER_POINTS['content-security-policy'],
      detail,
    });
    if (cspHas(csp, "'unsafe-inline'") && !cspHas(csp, 'strict-dynamic') && !cspHas(csp, 'nonce-') && !cspHas(csp, 'sha256-')) {
      warnings.push("CSP uses 'unsafe-inline' without nonce/hash/strict-dynamic — XSS can still inject inline script");
    }
    if (/(?:^|;)\s*(?:default-src|script-src)\s[^;]*\*/i.test(csp)) {
      warnings.push('CSP allows * in default-src or script-src');
    }
    if (!cspHas(csp, 'frame-ancestors') && !header(headers, 'x-frame-options')) {
      warnings.push('neither CSP frame-ancestors nor X-Frame-Options — clickjacking');
    }
  }

  const xfo = header(headers, 'x-frame-options');
  if (!xfo) {
    findings.push({
      feature: 'x-frame-options',
      points: 0,
      max: SECURITY_HEADER_POINTS['x-frame-options'],
      detail: 'missing — page can be framed (clickjacking)',
    });
  } else if (!/^(deny|sameorigin)$/i.test(xfo)) {
    findings.push({
      feature: 'x-frame-options',
      points: 0,
      max: SECURITY_HEADER_POINTS['x-frame-options'],
      detail: `${xfo} (only DENY or SAMEORIGIN score)`,
    });
  } else {
    findings.push({
      feature: 'x-frame-options',
      points: SECURITY_HEADER_POINTS['x-frame-options'],
      max: SECURITY_HEADER_POINTS['x-frame-options'],
      detail: xfo,
    });
  }

  const xcto = header(headers, 'x-content-type-options');
  if (!xcto) {
    findings.push({
      feature: 'x-content-type-options',
      points: 0,
      max: SECURITY_HEADER_POINTS['x-content-type-options'],
      detail: 'missing — MIME sniffing can turn attacker text into HTML/JS',
    });
  } else if (xcto.toLowerCase() !== 'nosniff') {
    findings.push({
      feature: 'x-content-type-options',
      points: 0,
      max: SECURITY_HEADER_POINTS['x-content-type-options'],
      detail: `${xcto} (only nosniff is valid)`,
    });
  } else {
    findings.push({
      feature: 'x-content-type-options',
      points: SECURITY_HEADER_POINTS['x-content-type-options'],
      max: SECURITY_HEADER_POINTS['x-content-type-options'],
      detail: xcto,
    });
  }

  const referrer = header(headers, 'referrer-policy');
  if (!referrer) {
    findings.push({
      feature: 'referrer-policy',
      points: 0,
      max: SECURITY_HEADER_POINTS['referrer-policy'],
      detail: 'missing — navigations leak full URLs (tokens/query) to third parties',
    });
  } else {
    const policies = referrer
      .split(',')
      .map((part) => part.trim().toLowerCase())
      .filter(Boolean);
    const ok = policies.every((policy) => SAFE_REFERRER_POLICIES.has(policy));
    findings.push({
      feature: 'referrer-policy',
      points: ok ? SECURITY_HEADER_POINTS['referrer-policy'] : 0,
      max: SECURITY_HEADER_POINTS['referrer-policy'],
      detail: ok ? referrer : `${referrer} (unsafe / unknown token)`,
    });
  }

  const permissions = header(headers, 'permissions-policy');
  if (!permissions) {
    findings.push({
      feature: 'permissions-policy',
      points: 0,
      max: SECURITY_HEADER_POINTS['permissions-policy'],
      detail: 'missing — camera/mic/geo/payment APIs are not locked down',
    });
  } else {
    findings.push({
      feature: 'permissions-policy',
      points: SECURITY_HEADER_POINTS['permissions-policy'],
      max: SECURITY_HEADER_POINTS['permissions-policy'],
      detail: permissions,
    });
    const compact = permissions.replace(/\s+/g, '');
    for (const feature of ['camera', 'microphone', 'geolocation']) {
      if (!compact.includes(`${feature}=()`)) {
        warnings.push(`Permissions-Policy does not disable ${feature}=()`);
      }
    }
  }

  const powered = header(headers, 'x-powered-by');
  if (powered) {
    warnings.push(`X-Powered-By leaks stack: ${powered}`);
  }
  const server = header(headers, 'server');
  if (server && /\/\d|[A-Za-z]+\d/.test(server)) {
    warnings.push(`Server discloses version: ${server}`);
  } else if (server) {
    warnings.push(`Server present (fingerprinting): ${server}`);
  }
  const xss = header(headers, 'x-xss-protection');
  if (xss && xss !== '0') {
    warnings.push(`deprecated X-XSS-Protection=${xss} (can introduce XSS in old IE; prefer omit or 0)`);
  }
  if (cspReportOnly && !csp) {
    warnings.push('CSP is report-only only — not enforced');
  }

  const score = findings.reduce((sum, item) => sum + item.points, 0);
  const percent = Math.round((score / SECURITY_HEADERS_MAX) * 100);
  const interesting = [
    'strict-transport-security',
    'content-security-policy',
    'content-security-policy-report-only',
    'x-frame-options',
    'x-content-type-options',
    'referrer-policy',
    'permissions-policy',
    'x-xss-protection',
    'x-powered-by',
    'server',
    'set-cookie',
    'access-control-allow-origin',
    'access-control-allow-credentials',
  ];
  const dumpLines = interesting.map((name) => `  ${name}: ${header(headers, name) || '<absent>'}`);

  return {
    url,
    status,
    score,
    percent,
    grade: gradeForPercent(percent),
    findings,
    warnings,
    headersDump: dumpLines.join('\n'),
  };
}

export function formatScan(scan: SecurityHeadersScan): string {
  const rows = scan.findings.map((item) => `  ${item.feature.padEnd(28)} ${String(item.points).padStart(2)}/${item.max}  ${item.detail}`).join('\n');
  const warns = scan.warnings.length === 0 ? '  (none)' : scan.warnings.map((w) => `  - ${w}`).join('\n');
  return [
    `securityheaders.com scan  ${scan.url}  HTTP ${scan.status}`,
    `grade ${scan.grade}  ${scan.score}/${SECURITY_HEADERS_MAX} (${scan.percent}%)  A+ requires ≥95%`,
    rows,
    'warnings:',
    warns,
    'raw:',
    scan.headersDump,
  ].join('\n');
}

export async function expectAPlusSecurityHeaders(response: Pick<APIResponse, 'url' | 'status' | 'headers'>, operation: string): Promise<SecurityHeadersScan> {
  const scan = scanSecurityHeaders(response.url(), response.status(), normalizeHeaders(response.headers()));
  const report = formatScan(scan);
  await test.info().attach(`${operation} securityheaders`, { body: report, contentType: 'text/plain' });

  for (const finding of scan.findings) {
    expect.soft(finding.points, `${operation}: ${finding.feature} — ${finding.detail}\n${report}`).toBe(finding.max);
  }
  expect.soft(scan.warnings, `${operation}: warnings\n${report}`).toEqual([]);
  expect(scan.grade, `${operation}: grade must be A+\n${report}`).toBe('A+');
  return scan;
}

export function inspectSetCookie(setCookie: string, operation: string): void {
  const lower = setCookie.toLowerCase();
  expect.soft(lower.includes('secure'), `${operation}: Set-Cookie must be Secure. ${setCookie}`).toBe(true);
  expect.soft(lower.includes('httponly'), `${operation}: Set-Cookie must be HttpOnly. ${setCookie}`).toBe(true);
  expect.soft(/samesite=(strict|lax)/i.test(setCookie), `${operation}: Set-Cookie must set SameSite=Strict|Lax. ${setCookie}`).toBe(true);
}
