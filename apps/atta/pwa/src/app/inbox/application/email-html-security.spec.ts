import { secureEmailHtml } from './email-html-security';

describe('email-html-security', () => {
  it('sanitizes dangerous HTML and hardens links', () => {
    const result = secureEmailHtml(`<div><script>alert(1)</script><a href="https://example.com">link</a><img src="https://tracker.example/pixel" onerror="alert(1)"></div>`, false);

    expect(result.sanitizedHtml).not.toContain('<script');
    expect(result.sanitizedHtml).toContain('target="_blank"');
    expect(result.sanitizedHtml).toContain('rel="noopener noreferrer"');
    expect(result.sanitizedHtml).toContain('data-blocked-src="https://tracker.example/pixel"');
    expect(result.blockedExternalImages).toBe(1);
  });

  it('keeps external image when user allows it', () => {
    const result = secureEmailHtml(`<img src="https://cdn.example/logo.png">`, true);

    expect(result.sanitizedHtml).toContain('src="https://cdn.example/logo.png"');
    expect(result.sanitizedHtml).not.toContain('data-blocked-src');
    expect(result.blockedExternalImages).toBe(0);
  });
});
