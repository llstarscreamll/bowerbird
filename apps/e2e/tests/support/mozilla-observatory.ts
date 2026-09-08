import { expect, test, type APIRequestContext, type APIResponse } from '@playwright/test';
import { normalizeHeaders, type HeaderMap } from './security-headers';

/** Mozilla HTTP Observatory: baseline 100, bonuses only if score after penalties ≥ 90. A+ is ≥ 100. */
export const OBSERVATORY_BASELINE = 100;
export const OBSERVATORY_BONUS_GATE = 90;
export const OBSERVATORY_CORS_ORIGIN = 'https://http-observatory.security.mozilla.org';
const HSTS_SIX_MONTHS = 15_552_000;

const DANGEROUSLY_BROAD = new Set(['ftp:', 'http:', 'https:', '*', 'http://*', 'http://*.*', 'https://*', 'https://*.*']);
const UNSAFE_INLINE = new Set(["'unsafe-inline'", 'data:']);
const PRIVATE_REFERRERS = new Set(['no-referrer', 'same-origin', 'strict-origin', 'strict-origin-when-cross-origin']);
const UNSAFE_REFERRERS = new Set(['origin', 'origin-when-cross-origin', 'unsafe-url', 'no-referrer-when-downgrade']);

export type ObservatoryTestName =
  | 'content-security-policy'
  | 'cookies'
  | 'cross-origin-resource-sharing'
  | 'redirection'
  | 'referrer-policy'
  | 'strict-transport-security'
  | 'subresource-integrity'
  | 'x-content-type-options'
  | 'x-frame-options'
  | 'cross-origin-opener-policy'
  | 'cross-origin-embedder-policy'
  | 'cross-origin-resource-policy';

export type ObservatoryFinding = {
  test: ObservatoryTestName;
  result: string;
  modifier: number;
  pass: boolean;
  detail: string;
};

export type ObservatoryScan = {
  url: string;
  status: number;
  score: number;
  grade: string;
  findings: ObservatoryFinding[];
  warnings: string[];
  report: string;
};

export type HttpRedirectProbe = {
  reached: boolean;
  status?: number;
  location?: string;
};

export type CorsProbe = {
  acao: string;
  credentials: string;
  requestOrigin: string;
};

const header = (headers: HeaderMap, name: string): string => headers[name.toLowerCase()]?.trim() ?? '';

const parseHstsMaxAge = (value: string): number | null => {
  const match = value.match(/(?:^|;)\s*max-age=(\d+)/i);
  return match ? Number(match[1]) : null;
};

const cspMap = (csp: string): Map<string, Set<string>> => {
  const map = new Map<string, Set<string>>();
  for (const raw of csp.split(';')) {
    const tokens = raw.trim().split(/\s+/).filter(Boolean);
    if (tokens.length === 0) {
      continue;
    }
    const name = tokens[0].toLowerCase();
    const values = map.get(name) ?? new Set<string>();
    for (const token of tokens.slice(1)) {
      values.add(token);
    }
    map.set(name, values);
  }
  return map;
};

const hasNonceOrHash = (sources: Set<string>): boolean => [...sources].some((source) => ["'sha256-", "'sha384-", "'sha512-", "'nonce-"].some((prefix) => source.toLowerCase().startsWith(prefix)));

const gradeForScore = (score: number): string => {
  if (score >= 100) {
    return 'A+';
  }
  if (score >= 90) {
    return 'A';
  }
  if (score >= 85) {
    return 'A-';
  }
  if (score >= 80) {
    return 'B+';
  }
  if (score >= 70) {
    return 'B';
  }
  if (score >= 65) {
    return 'B-';
  }
  if (score >= 60) {
    return 'C+';
  }
  if (score >= 50) {
    return 'C';
  }
  if (score >= 45) {
    return 'C-';
  }
  if (score >= 40) {
    return 'D+';
  }
  if (score >= 30) {
    return 'D';
  }
  if (score >= 25) {
    return 'D-';
  }
  return 'F';
};

const finding = (name: ObservatoryTestName, result: string, modifier: number, pass: boolean, detail: string): ObservatoryFinding => ({ test: name, result, modifier, pass, detail });

const scoreCsp = (headers: HeaderMap): ObservatoryFinding => {
  const csp = header(headers, 'content-security-policy');
  const reportOnly = header(headers, 'content-security-policy-report-only');
  if (!csp) {
    return finding(
      'content-security-policy',
      reportOnly ? 'csp-not-implemented-but-reporting-enabled' : 'csp-not-implemented',
      -25,
      false,
      reportOnly ? `only Report-Only: ${reportOnly}` : 'missing — XSS can load arbitrary script',
    );
  }

  const parsed = cspMap(csp);
  const scriptSrc = new Set(parsed.get('script-src') ?? parsed.get('default-src') ?? ['*']);
  const objectSrc = new Set(parsed.get('object-src') ?? parsed.get('default-src') ?? ['*']);
  const styleSrc = new Set(parsed.get('style-src') ?? parsed.get('default-src') ?? ['*']);
  if (hasNonceOrHash(scriptSrc)) {
    scriptSrc.delete("'unsafe-inline'");
  }
  if (hasNonceOrHash(styleSrc)) {
    styleSrc.delete("'unsafe-inline'");
  }

  const scriptUnsafe = [...scriptSrc].some((src) => DANGEROUSLY_BROAD.has(src) || UNSAFE_INLINE.has(src));
  const objectBroad = [...objectSrc].some((src) => DANGEROUSLY_BROAD.has(src));
  if (scriptUnsafe || objectBroad) {
    return finding('content-security-policy', 'csp-implemented-with-unsafe-inline', -20, false, `${csp} — unsafe-inline/data:/https: in script-src or unrestricted object-src`);
  }
  if (scriptSrc.has("'unsafe-eval'") || styleSrc.has("'unsafe-eval'")) {
    return finding('content-security-policy', 'csp-implemented-with-unsafe-eval', -10, false, csp);
  }
  if ([...styleSrc].some((src) => DANGEROUSLY_BROAD.has(src) || UNSAFE_INLINE.has(src))) {
    return finding('content-security-policy', 'csp-implemented-with-unsafe-inline-in-style-src-only', 0, true, `${csp} — style-src still allows XSS via CSS / injection of style`);
  }
  const defaultSrc = parsed.get('default-src');
  if (defaultSrc?.size === 1 && defaultSrc.has("'none'")) {
    return finding('content-security-policy', 'csp-implemented-with-no-unsafe-default-src-none', 10, true, csp);
  }
  return finding('content-security-policy', 'csp-implemented-with-no-unsafe', 5, true, csp);
};

const cookieNames = (raw: string): string => raw.split('=', 1)[0] ?? 'cookie';

const scoreCookies = (setCookieValues: string[], hstsPass: boolean): ObservatoryFinding => {
  if (setCookieValues.length === 0) {
    return finding('cookies', 'cookies-not-found', 0, true, 'no Set-Cookie on this response');
  }

  const names = setCookieValues.map(cookieNames).join(', ');
  let worst = finding('cookies', 'cookies-secure-with-httponly-sessions-and-samesite', 5, true, names);
  const worse = (result: string, modifier: number, pass: boolean, detail: string) => {
    if (modifier < worst.modifier) {
      worst = finding('cookies', result, modifier, pass, detail);
    }
  };

  let missingSameSite = false;
  for (const raw of setCookieValues) {
    const lower = raw.toLowerCase();
    const name = raw.split('=', 1)[0] ?? '';
    const session = /login|sess/i.test(name);
    const csrf = /csrf/i.test(name);
    const secure = /(?:^|;)\s*secure(?:;|$)/i.test(raw);
    const httpOnly = /(?:^|;)\s*httponly(?:;|$)/i.test(raw);
    const sameSiteMatch = lower.match(/samesite=([^;]+)/i);
    const sameSite = sameSiteMatch?.[1]?.trim() ?? '';

    if (sameSite && !['strict', 'lax'].includes(sameSite) && !(sameSite === 'none' && secure)) {
      worse('cookies-samesite-flag-invalid', -20, false, names);
    }
    if (sameSite === 'none' && !secure) {
      worse('cookies-samesite-flag-invalid', -20, false, names);
    }
    if (!secure) {
      worse(hstsPass ? 'cookies-without-secure-flag-but-protected-by-hsts' : 'cookies-without-secure-flag', hstsPass ? -5 : -20, false, names);
    }
    if (csrf && !sameSite) {
      worse('cookies-anticsrf-without-samesite-flag', -20, false, names);
    }
    if (session && !secure) {
      worse(hstsPass ? 'cookies-session-without-secure-flag-but-protected-by-hsts' : 'cookies-session-without-secure-flag', hstsPass ? -10 : -40, false, names);
    }
    if (session && !httpOnly) {
      worse('cookies-session-without-httponly-flag', -30, false, names);
    }
    if (!sameSite) {
      missingSameSite = true;
    }
  }

  if (worst.modifier === 5 && missingSameSite) {
    return finding('cookies', 'cookies-secure-with-httponly-sessions', 0, true, names);
  }
  return worst;
};

const scoreCors = (cors: CorsProbe | undefined): ObservatoryFinding => {
  if (!cors || !cors.acao) {
    return finding('cross-origin-resource-sharing', 'cross-origin-resource-sharing-not-implemented', 0, true, 'no ACAO');
  }
  const acao = cors.acao.trim();
  if (acao === '*') {
    return finding('cross-origin-resource-sharing', 'cross-origin-resource-sharing-implemented-with-public-access', 0, true, 'Access-Control-Allow-Origin: * — any origin can read the response');
  }
  if (cors.requestOrigin && acao === cors.requestOrigin && cors.credentials.toLowerCase() === 'true') {
    return finding(
      'cross-origin-resource-sharing',
      'cross-origin-resource-sharing-implemented-with-universal-access',
      -50,
      false,
      `reflects Origin ${cors.requestOrigin} with credentials — any site can steal credentialed responses`,
    );
  }
  return finding('cross-origin-resource-sharing', 'cross-origin-resource-sharing-implemented-with-restricted-access', 0, true, acao);
};

const scoreRedirection = (httpsUrl: string, probe: HttpRedirectProbe | undefined): ObservatoryFinding => {
  if (!probe || !probe.reached) {
    return finding('redirection', 'redirection-not-needed-no-http', 0, true, 'HTTP port not reachable');
  }
  const location = probe.location ?? '';
  const redirectedToHttps = /^https:\/\//i.test(location);
  if ((probe.status ?? 0) < 300 || (probe.status ?? 0) >= 400) {
    if (redirectedToHttps) {
      return finding('redirection', 'redirection-to-https', 0, true, location);
    }
    return finding('redirection', 'redirection-missing', -20, false, `HTTP ${probe.status} without HTTPS Location — SSL stripping / MITM can keep the user on HTTP`);
  }
  if (!redirectedToHttps) {
    return finding('redirection', 'redirection-not-to-https', -20, false, `Location: ${location || '<empty>'}`);
  }
  try {
    const fromHost = new URL(httpsUrl.replace(/^https:/, 'http:')).hostname;
    const toHost = new URL(location, httpsUrl).hostname;
    if (fromHost !== toHost) {
      return finding('redirection', 'redirection-off-host-from-http', -5, false, `${fromHost} → ${toHost} — first hop must stay on the same host so HSTS can stick`);
    }
  } catch {
    return finding('redirection', 'redirection-missing', -20, false, location);
  }
  return finding('redirection', 'redirection-to-https', 0, true, location);
};

const scoreReferrer = (headers: HeaderMap): ObservatoryFinding => {
  const value = header(headers, 'referrer-policy');
  if (!value) {
    return finding('referrer-policy', 'referrer-policy-not-implemented', 0, true, 'missing');
  }
  const tokens = value
    .split(',')
    .map((part) => part.trim().toLowerCase())
    .filter(Boolean);
  if (tokens.every((token) => PRIVATE_REFERRERS.has(token))) {
    return finding('referrer-policy', 'referrer-policy-private', 5, true, value);
  }
  if (tokens.some((token) => UNSAFE_REFERRERS.has(token))) {
    return finding('referrer-policy', 'referrer-policy-unsafe', -5, false, value);
  }
  return finding('referrer-policy', 'referrer-policy-header-invalid', -5, false, value);
};

const scoreHsts = (url: string, headers: HeaderMap): ObservatoryFinding => {
  if (!url.toLowerCase().startsWith('https://')) {
    return finding('strict-transport-security', 'hsts-not-implemented-no-https', -20, false, url);
  }
  const hsts = header(headers, 'strict-transport-security');
  if (!hsts) {
    return finding('strict-transport-security', 'hsts-not-implemented', -20, false, 'missing');
  }
  if (hsts.includes(',')) {
    return finding('strict-transport-security', 'hsts-header-invalid', -20, false, hsts);
  }
  const maxAge = parseHstsMaxAge(hsts);
  if (maxAge === null) {
    return finding('strict-transport-security', 'hsts-header-invalid', -20, false, hsts);
  }
  if (maxAge < HSTS_SIX_MONTHS) {
    return finding('strict-transport-security', 'hsts-implemented-max-age-less-than-six-months', -10, false, hsts);
  }
  return finding('strict-transport-security', 'hsts-implemented-max-age-at-least-six-months', 0, true, hsts);
};

const collectScripts = (html: string): Array<{ src: string; integrity: string }> => {
  const scripts: Array<{ src: string; integrity: string }> = [];
  const re = /<script\b([^>]*)>/gi;
  let match: RegExpExecArray | null;
  while ((match = re.exec(html)) !== null) {
    const attrs = match[1] ?? '';
    const src = attrs.match(/\bsrc\s*=\s*["']([^"']+)["']/i)?.[1];
    if (!src) {
      continue;
    }
    const integrity = attrs.match(/\bintegrity\s*=\s*["']([^"']+)["']/i)?.[1] ?? '';
    scripts.push({ src, integrity });
  }
  return scripts;
};

const scoreSri = (url: string, contentType: string, html: string | undefined): ObservatoryFinding => {
  const mime = contentType.split(';', 1)[0]?.trim().toLowerCase() ?? '';
  if (mime && !mime.includes('html')) {
    return finding('subresource-integrity', 'sri-not-implemented-response-not-html', 0, true, mime || 'non-html');
  }
  if (!html) {
    return finding('subresource-integrity', 'sri-not-implemented-response-not-html', 0, true, 'no body');
  }
  const scripts = collectScripts(html);
  if (scripts.length === 0) {
    return finding('subresource-integrity', 'sri-not-implemented-but-no-scripts-loaded', 0, true, 'no script src');
  }

  let foreign = false;
  let worst: ObservatoryFinding | null = null;
  const pageHost = (() => {
    try {
      return new URL(url).hostname;
    } catch {
      return '';
    }
  })();

  for (const script of scripts) {
    const relativeProtocol = /^\/\//.test(script.src);
    const absolute = /^https?:\/\//i.test(script.src);
    const sameOrigin = !relativeProtocol && !absolute;
    let sameSite = sameOrigin;
    let scheme = '';
    if (absolute) {
      try {
        const parsed = new URL(script.src);
        scheme = parsed.protocol;
        sameSite = parsed.hostname === pageHost || parsed.hostname.endsWith(`.${pageHost.split('.').slice(-2).join('.')}`);
      } catch {
        sameSite = false;
      }
    }
    const foreignOrigin = relativeProtocol || (absolute && !sameSite);
    if (foreignOrigin) {
      foreign = true;
      const secureScheme = scheme === 'https:';
      if (script.integrity && !secureScheme) {
        worst = finding('subresource-integrity', 'sri-implemented-but-external-scripts-not-loaded-securely', -20, false, script.src);
      } else if (!script.integrity && secureScheme) {
        worst = finding('subresource-integrity', 'sri-not-implemented-but-external-scripts-loaded-securely', -5, false, `${script.src} — CDN compromise executes as the site`);
      } else if (!script.integrity) {
        worst = finding('subresource-integrity', 'sri-not-implemented-and-external-scripts-not-loaded-securely', -50, false, script.src);
      }
    }
  }

  if (worst) {
    return worst;
  }
  if (foreign) {
    return finding('subresource-integrity', 'sri-implemented-and-external-scripts-loaded-securely', 5, true, 'external scripts have integrity');
  }
  const sameOriginIntegrity = scripts.every((script) => script.integrity);
  if (sameOriginIntegrity) {
    return finding('subresource-integrity', 'sri-implemented-and-all-scripts-loaded-securely', 5, true, 'same-origin scripts have integrity');
  }
  return finding('subresource-integrity', 'sri-not-implemented-but-all-scripts-loaded-from-secure-origin', 0, true, scripts.map((script) => script.src).join(', '));
};

const scoreXcto = (headers: HeaderMap): ObservatoryFinding => {
  const value = header(headers, 'x-content-type-options');
  if (!value) {
    return finding('x-content-type-options', 'x-content-type-options-not-implemented', -5, false, 'missing');
  }
  if (value.toLowerCase() !== 'nosniff') {
    return finding('x-content-type-options', 'x-content-type-options-header-invalid', -5, false, value);
  }
  return finding('x-content-type-options', 'x-content-type-options-nosniff', 0, true, value);
};

const scoreXfo = (headers: HeaderMap): ObservatoryFinding => {
  const csp = header(headers, 'content-security-policy');
  const hasFrameAncestors = /(?:^|;)\s*frame-ancestors\b/i.test(csp);
  const xfo = header(headers, 'x-frame-options');
  if (hasFrameAncestors) {
    return finding('x-frame-options', 'x-frame-options-implemented-via-csp', 5, true, 'CSP frame-ancestors');
  }
  if (!xfo) {
    return finding('x-frame-options', 'x-frame-options-not-implemented', -20, false, 'missing — clickjacking');
  }
  if (/^(deny|sameorigin)$/i.test(xfo)) {
    return finding('x-frame-options', 'x-frame-options-sameorigin-or-deny', 5, true, xfo);
  }
  if (/^allow-from\b/i.test(xfo)) {
    return finding('x-frame-options', 'x-frame-options-allow-from-origin', 0, true, xfo);
  }
  return finding('x-frame-options', 'x-frame-options-header-invalid', -20, false, xfo);
};

const scoreCoop = (headers: HeaderMap): ObservatoryFinding => {
  const value = header(headers, 'cross-origin-opener-policy').toLowerCase();
  if (!value) {
    return finding('cross-origin-opener-policy', 'coop-not-implemented', 0, true, 'missing — window.opener / XS-Leaks');
  }
  if (value === 'same-origin') {
    return finding('cross-origin-opener-policy', 'coop-implemented-with-same-origin', 10, true, value);
  }
  if (value === 'same-origin-allow-popups') {
    return finding('cross-origin-opener-policy', 'coop-implemented-with-same-origin-allow-popups', 10, true, value);
  }
  if (value === 'noopener-allow-popups') {
    return finding('cross-origin-opener-policy', 'coop-implemented-with-noopener-allow-popups', 10, true, value);
  }
  if (value === 'unsafe-none') {
    return finding('cross-origin-opener-policy', 'coop-implemented-with-unsafe-none', 0, true, value);
  }
  return finding('cross-origin-opener-policy', 'coop-header-invalid', -5, false, value);
};

const scoreCoep = (headers: HeaderMap): ObservatoryFinding => {
  const value = header(headers, 'cross-origin-embedder-policy').toLowerCase();
  if (!value) {
    return finding('cross-origin-embedder-policy', 'coep-not-implemented', 0, true, 'missing — no cross-origin isolation');
  }
  if (value === 'require-corp') {
    return finding('cross-origin-embedder-policy', 'coep-implemented-with-require-corp', 10, true, value);
  }
  if (value === 'credentialless') {
    return finding('cross-origin-embedder-policy', 'coep-implemented-with-credentialless', 10, true, value);
  }
  if (value === 'unsafe-none') {
    return finding('cross-origin-embedder-policy', 'coep-implemented-with-unsafe-none', 0, true, value);
  }
  return finding('cross-origin-embedder-policy', 'coep-header-invalid', -5, false, value);
};

const scoreCorp = (headers: HeaderMap): ObservatoryFinding => {
  const value = header(headers, 'cross-origin-resource-policy').toLowerCase();
  if (!value) {
    return finding('cross-origin-resource-policy', 'cross-origin-resource-policy-not-implemented', 0, true, 'missing — Spectre / no-cors reads');
  }
  if (value === 'same-origin') {
    return finding('cross-origin-resource-policy', 'cross-origin-resource-policy-implemented-with-same-origin', 10, true, value);
  }
  if (value === 'same-site') {
    return finding('cross-origin-resource-policy', 'cross-origin-resource-policy-implemented-with-same-site', 10, true, value);
  }
  if (value === 'cross-origin') {
    return finding('cross-origin-resource-policy', 'cross-origin-resource-policy-implemented-with-cross-origin', 0, true, value);
  }
  return finding('cross-origin-resource-policy', 'cross-origin-resource-policy-header-invalid', -5, false, value);
};

export function scanMozillaObservatory(input: {
  url: string;
  status: number;
  headers: HeaderMap;
  html?: string;
  setCookies?: string[];
  cors?: CorsProbe;
  httpRedirect?: HttpRedirectProbe;
}): ObservatoryScan {
  const headers = normalizeHeaders(input.headers);
  const hsts = scoreHsts(input.url, headers);
  const findings: ObservatoryFinding[] = [
    scoreCsp(headers),
    scoreCookies(input.setCookies ?? [], hsts.pass),
    scoreCors(input.cors),
    scoreRedirection(input.url, input.httpRedirect),
    scoreReferrer(headers),
    hsts,
    scoreSri(input.url, header(headers, 'content-type'), input.html),
    scoreXcto(headers),
    scoreXfo(headers),
    scoreCoop(headers),
    scoreCoep(headers),
    scoreCorp(headers),
  ];

  const penalties = findings.filter((item) => item.modifier < 0);
  let score = OBSERVATORY_BASELINE + penalties.reduce((sum, item) => sum + item.modifier, 0);
  score = Math.max(0, score);
  if (score >= OBSERVATORY_BONUS_GATE) {
    score += findings.filter((item) => item.modifier > 0).reduce((sum, item) => sum + item.modifier, 0);
  }

  const warnings: string[] = [];
  const csp = findings.find((item) => item.test === 'content-security-policy');
  if (csp?.result === 'csp-implemented-with-unsafe-inline-in-style-src-only') {
    warnings.push('CSP extra credit blocked: style-src allows unsafe-inline/data:/https: (Observatory 0, not +5/+10)');
  }
  const coop = findings.find((item) => item.test === 'cross-origin-opener-policy');
  if (coop && coop.modifier <= 0) {
    warnings.push('COOP missing/weak — Observatory +10 for same-origin (XS-Leaks / window.opener)');
  }
  const coep = findings.find((item) => item.test === 'cross-origin-embedder-policy');
  if (coep && coep.modifier <= 0) {
    warnings.push('COEP missing/weak — Observatory +10 for require-corp or credentialless');
  }
  const corp = findings.find((item) => item.test === 'cross-origin-resource-policy');
  if (corp && corp.modifier <= 0) {
    warnings.push('CORP missing/weak — Observatory +10 for same-origin/same-site (no-cors Spectre reads)');
  }
  const xss = header(headers, 'x-xss-protection');
  if (xss && xss !== '0') {
    warnings.push(`deprecated X-XSS-Protection=${xss} can enable XSS in old IE`);
  }

  const grade = gradeForScore(score);
  const rows = findings.map((item) => `  ${item.test.padEnd(32)} ${String(item.modifier).padStart(3)}  ${item.pass ? 'pass' : 'FAIL'}  ${item.result}  ${item.detail}`).join('\n');
  const report = [
    `Mozilla HTTP Observatory  ${input.url}  HTTP ${input.status}`,
    `grade ${grade}  ${score}  A+ requires ≥100 (bonuses only if penalties leave ≥90)`,
    rows,
    'warnings:',
    warnings.length === 0 ? '  (none)' : warnings.map((item) => `  - ${item}`).join('\n'),
  ].join('\n');

  return { url: input.url, status: input.status, score, grade, findings, warnings, report };
}

export function setCookiesFromResponse(response: Pick<APIResponse, 'headersArray'>): string[] {
  return response
    .headersArray()
    .filter((item) => item.name.toLowerCase() === 'set-cookie')
    .map((item) => item.value);
}

export async function probeHttpRedirect(request: APIRequestContext, httpsUrl: string): Promise<HttpRedirectProbe> {
  const httpUrl = httpsUrl.replace(/^https:/i, 'http:');
  try {
    const response = await request.get(httpUrl, { maxRedirects: 0 });
    return {
      reached: true,
      status: response.status(),
      location: response.headers()['location'] ?? '',
    };
  } catch {
    return { reached: false };
  }
}

export async function probeCors(request: APIRequestContext, url: string, origin: string): Promise<CorsProbe> {
  const response = await request.fetch(url, {
    method: 'OPTIONS',
    headers: {
      Origin: origin,
      'Access-Control-Request-Method': 'GET',
    },
  });
  const headers = normalizeHeaders(response.headers());
  return {
    acao: header(headers, 'access-control-allow-origin'),
    credentials: header(headers, 'access-control-allow-credentials'),
    requestOrigin: origin,
  };
}

export async function expectObservatoryAPlus(scan: ObservatoryScan, operation: string): Promise<void> {
  await test.info().attach(`${operation} mozilla-observatory`, { body: scan.report, contentType: 'text/plain' });
  for (const item of scan.findings) {
    expect.soft(item.pass, `${operation}: ${item.test} ${item.result} (${item.modifier})\n${scan.report}`).toBe(true);
    expect.soft(item.modifier, `${operation}: ${item.test} must not penalize\n${scan.report}`).toBeGreaterThanOrEqual(0);
  }
  expect.soft(scan.warnings, `${operation}: missed Observatory extras\n${scan.report}`).toEqual([]);
  expect(scan.grade, `${operation}: grade must be A+\n${scan.report}`).toBe('A+');
}
