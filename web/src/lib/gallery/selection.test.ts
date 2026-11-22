import { describe, expect, it } from 'vitest';

import { selectRange, toggleKey } from '@/lib/gallery/selection';

describe('toggleKey', () => {
  it('adds a key that is not chosen', () => {
    expect(toggleKey(['a'], 'b')).toEqual(['a', 'b']);
  });

  it('removes a chosen key', () => {
    expect(toggleKey(['a', 'b'], 'a')).toEqual(['b']);
  });
});

describe('selectRange', () => {
  const keys = ['a', 'b', 'c', 'd', 'e'];

  it('adds every key from the anchor to the target', () => {
    expect(selectRange(['a'], keys, 'b', 'd')).toEqual(['a', 'b', 'c', 'd']);
  });

  it('adds the keys in both directions', () => {
    expect(selectRange([], keys, 'd', 'b')).toEqual(['b', 'c', 'd']);
  });

  it('adds the target alone without an anchor', () => {
    expect(selectRange(['a'], keys, null, 'c')).toEqual(['a', 'c']);
  });
});
