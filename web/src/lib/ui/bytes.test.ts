import { describe, expect, it } from 'vitest';

import { formatBytes } from '@/lib/ui/bytes';

describe('formatBytes', () => {
  it('keeps a small number in bytes', () => {
    expect(formatBytes(512, 'en')).toBe('512 byte');
  });

  it('picks the largest unit that keeps the value at one or above', () => {
    expect(formatBytes(1_500_000, 'en')).toBe('1.5 MB');
    expect(formatBytes(120_000_000_000, 'en')).toBe('120 GB');
  });

  it('formats the number for the locale', () => {
    expect(formatBytes(1_500_000, 'de')).toBe('1,5 MB');
  });

  it('stops at the largest unit', () => {
    expect(formatBytes(2_000_000_000_000_000, 'en')).toBe('2,000 TB');
  });
});
