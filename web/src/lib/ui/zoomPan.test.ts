import { describe, expect, it } from 'vitest';

import {
  clampOffset,
  clampScale,
  distance,
  midpoint,
  unzoomed,
  wheelScale,
  zoomAround,
} from '@/lib/ui/zoomPan';

// The element is 200 by 100 pixels
const size = {
  width: 200,
  height: 100,
};

describe('unzoomed', () => {
  it('returns the normal scale without an offset', () => {
    expect(unzoomed()).toEqual({
      scale: 1,
      x: 0,
      y: 0,
    });
  });
});

describe('clampScale', () => {
  it('keeps a scale within the bounds', () => {
    expect(clampScale(2, 4)).toBe(2);
  });

  it('raises a scale below the normal one', () => {
    expect(clampScale(0.5, 4)).toBe(1);
  });

  it('lowers a scale above the maximum', () => {
    expect(clampScale(6, 4)).toBe(4);
  });
});

describe('clampOffset', () => {
  it('keeps an offset within the overflow', () => {
    expect(clampOffset(30, 2, 200)).toBe(30);
  });

  it('limits an offset to half of the overflow', () => {
    expect(clampOffset(150, 2, 200)).toBe(100);
    expect(clampOffset(-150, 2, 200)).toBe(-100);
  });

  it('allows no offset at the normal scale', () => {
    expect(clampOffset(50, 1, 200)).toBe(0);
  });
});

describe('zoomAround', () => {
  it('keeps the point under the position in place', () => {
    const next = zoomAround(
      {
        scale: 1,
        x: 0,
        y: 0,
      },
      2,
      {
        x: 40,
        y: 20,
      },
      size,
    );

    expect(next).toEqual({
      scale: 2,
      x: -40,
      y: -20,
    });
  });

  it('keeps the edges outside the element', () => {
    const next = zoomAround(
      {
        scale: 1,
        x: 0,
        y: 0,
      },
      2,
      {
        x: 100,
        y: 50,
      },
      size,
    );

    expect(next).toEqual({
      scale: 2,
      x: -100,
      y: -50,
    });
  });

  it('resets the offset at the normal scale', () => {
    const next = zoomAround(
      {
        scale: 2,
        x: -40,
        y: -20,
      },
      1,
      {
        x: 10,
        y: 10,
      },
      size,
    );

    expect(next).toEqual({
      scale: 1,
      x: 0,
      y: 0,
    });
  });
});

describe('wheelScale', () => {
  it('zooms in for a wheel step up', () => {
    expect(wheelScale(1, -100, 1.5, 4)).toBe(1.5);
  });

  it('zooms out for a wheel step down', () => {
    expect(wheelScale(3, 100, 1.5, 4)).toBe(2);
  });

  it('stays within the bounds', () => {
    expect(wheelScale(1, 100, 1.5, 4)).toBe(1);
    expect(wheelScale(4, -100, 1.5, 4)).toBe(4);
  });
});

describe('distance', () => {
  it('returns the distance between two points', () => {
    expect(
      distance(
        {
          x: 0,
          y: 0,
        },
        {
          x: 3,
          y: 4,
        },
      ),
    ).toBe(5);
  });
});

describe('midpoint', () => {
  it('returns the point halfway between two points', () => {
    expect(
      midpoint(
        {
          x: 0,
          y: 10,
        },
        {
          x: 20,
          y: 30,
        },
      ),
    ).toEqual({
      x: 10,
      y: 20,
    });
  });
});
