import { isMergeInternalCodeInherited, previewMergeInternalCode } from './catalog.model';

describe('previewMergeInternalCode', () => {
  it('uses survivor code when present', () => {
    expect(
      previewMergeInternalCode(
        { internal_code: 'A' },
        [
          { id: '1', internal_code: 'A' },
          { id: '2', internal_code: 'B' },
        ],
        '2',
      ),
    ).toBe('A');
  });

  it('inherits the only distinct code when survivor has none', () => {
    expect(
      previewMergeInternalCode(
        { internal_code: null },
        [
          { id: '1', internal_code: null },
          { id: '2', internal_code: 'B' },
        ],
        '',
      ),
    ).toBe('B');
  });

  it('uses chosen code when multiple distinct codes exist', () => {
    expect(
      previewMergeInternalCode(
        { internal_code: null },
        [
          { id: '1', internal_code: 'A' },
          { id: '2', internal_code: 'B' },
        ],
        '2',
      ),
    ).toBe('B');
  });

  it('returns null when survivor has no code and none exist in the group', () => {
    expect(previewMergeInternalCode({ internal_code: null }, [{ id: '1', internal_code: null }], '')).toBeNull();
  });
});

describe('isMergeInternalCodeInherited', () => {
  it('is true when survivor had no code and a single source code is applied', () => {
    expect(isMergeInternalCodeInherited({ internal_code: null }, [{ internal_code: 'B' }], 'B')).toBe(true);
  });

  it('is false when survivor already had the code', () => {
    expect(isMergeInternalCodeInherited({ internal_code: 'A' }, [{ internal_code: 'A' }], 'A')).toBe(false);
  });
});
