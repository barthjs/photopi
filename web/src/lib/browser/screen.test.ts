import { describe, expect, it } from 'vitest';

import { screenPixels } from '@/lib/browser/screen';

describe('screenPixels', () => {
  it('returns the size of the view in device pixels', () => {
    expect(
      screenPixels({
        innerWidth: 960,
        innerHeight: 540,
        devicePixelRatio: 2,
      }),
    ).toEqual({
      screenWidth: 1920,
      screenHeight: 1080,
    });
  });

  it('takes a missing pixel ratio as one', () => {
    expect(
      screenPixels({
        innerWidth: 800,
        innerHeight: 480,
        devicePixelRatio: 0,
      }),
    ).toEqual({
      screenWidth: 800,
      screenHeight: 480,
    });
  });
});
