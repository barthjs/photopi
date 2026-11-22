import { describe, expect, it } from 'vitest';

import {
  paperSize,
  parseRatio,
  roundMm,
  slotBounds,
  snapSlotToWholeMm,
} from '@/lib/layout/layoutGeometry';

describe('roundMm', () => {
  it('rounds to whole millimeters', () => {
    const rounded = roundMm(12.6);

    expect(rounded).toBe(13);
  });
});

describe('snapSlotToWholeMm', () => {
  it('rounds position and size to whole millimeters', () => {
    const slot = {
      x: 0.1004,
      y: 0.2004,
      w: 0.3004,
      h: 0.4004,
      locked: false,
    };
    const paper = {
      widthMM: 100,
      heightMM: 100,
    };

    snapSlotToWholeMm(slot, paper, 0.05);

    expect(slot.x).toBeCloseTo(0.1);
    expect(slot.y).toBeCloseTo(0.2);
    expect(slot.w).toBeCloseTo(0.3);
    expect(slot.h).toBeCloseTo(0.4);
  });

  it('keeps the slot inside the page', () => {
    const slot = {
      x: 0.9,
      y: 0,
      w: 0.2,
      h: 0.5,
      locked: false,
    };
    const paper = {
      widthMM: 100,
      heightMM: 100,
    };

    snapSlotToWholeMm(slot, paper, 0.05);

    expect(slot.x + slot.w).toBeLessThanOrEqual(1);
  });
});

describe('paperSize', () => {
  it('returns the paper in millimeters', () => {
    const size = paperSize({
      widthMM: 100,
      heightMM: 150,
    });

    expect(size).toEqual({
      width: 100,
      height: 150,
    });
  });
});

describe('slotBounds', () => {
  it('places the slot on the paper in millimeters', () => {
    const slot = {
      x: 0.25,
      y: 0.5,
      w: 0.5,
      h: 0.25,
      locked: false,
    };

    const bounds = slotBounds(slot, {
      widthMM: 100,
      heightMM: 200,
    });

    expect(bounds).toEqual({
      minX: 25,
      maxX: 75,
      minY: 100,
      maxY: 150,
    });
  });
});

describe('parseRatio', () => {
  it.each([
    {
      name: 'reads a ratio with a colon',
      input: '4:3',
      expected: 4 / 3,
    },
    {
      name: 'reads a ratio with a slash',
      input: '16/9',
      expected: 16 / 9,
    },
    {
      name: 'reads a single number',
      input: '1.5',
      expected: 1.5,
    },
  ])('$name', ({ input, expected }) => {
    const ratio = parseRatio(input);

    expect(ratio).toBe(expected);
  });

  it.each([
    {
      name: 'rejects a ratio without width',
      input: '0:1',
    },
    {
      name: 'rejects text that is no ratio',
      input: 'wide',
    },
  ])('$name', ({ input }) => {
    const ratio = parseRatio(input);

    expect(ratio).toBeNull();
  });
});
