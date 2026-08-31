import { describe, expect, it, vi } from 'vitest';
import type { SystemNotice } from './system-notice.port';
import { SystemNoticesOrchestrator } from './system-notices.orchestrator';

function fakeNotice(overrides: Partial<SystemNotice> & Pick<SystemNotice, 'id' | 'priority' | 'scope'>): SystemNotice {
  return {
    canShow: () => true,
    show: vi.fn(),
    dismiss: vi.fn(),
    ...overrides,
  };
}

describe('SystemNoticesOrchestrator', () => {
  it('shows only the highest-priority eligible notice', () => {
    const low = fakeNotice({ id: 'low', priority: 10, scope: 'tenant' });
    const high = fakeNotice({ id: 'high', priority: 100, scope: 'tenant' });
    const orchestrator = new SystemNoticesOrchestrator([low, high]);

    orchestrator.tryPresent('tenant');

    expect(high.show).toHaveBeenCalledOnce();
    expect(low.show).not.toHaveBeenCalled();
    expect(orchestrator.activeId).toBe('high');
  });

  it('shows only one notice at a time', () => {
    const first = fakeNotice({ id: 'first', priority: 100, scope: 'global' });
    const second = fakeNotice({ id: 'second', priority: 50, scope: 'global' });
    const orchestrator = new SystemNoticesOrchestrator([first, second]);

    orchestrator.tryPresent('global');
    orchestrator.tryPresent('global');

    expect(first.show).toHaveBeenCalledOnce();
    expect(second.show).not.toHaveBeenCalled();
  });

  it('continues when show() throws', () => {
    const broken = fakeNotice({
      id: 'broken',
      priority: 100,
      scope: 'tenant',
      show: () => {
        throw new Error('boom');
      },
    });
    const fallback = fakeNotice({ id: 'fallback', priority: 50, scope: 'tenant' });
    const orchestrator = new SystemNoticesOrchestrator([broken, fallback]);

    orchestrator.tryPresent('tenant');

    expect(fallback.show).toHaveBeenCalledOnce();
    expect(orchestrator.activeId).toBe('fallback');
  });
});
