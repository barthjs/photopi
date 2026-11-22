import { describe, expect, it } from 'vitest';

import {
  inPeriod,
  parentPeriod,
  periodKey,
  periodOffset,
  periodStart,
  timelineColumns,
  timelineRows,
} from '@/lib/gallery/timeline';

describe('periodKey', () => {
  const date = new Date(2026, 10, 5, 12);

  it.each([
    {
      zoom: 'year' as const,
      key: '2026',
    },
    {
      zoom: 'month' as const,
      key: '2026-11',
    },
    {
      zoom: 'week' as const,
      key: '2026-W45',
    },
    {
      zoom: 'day' as const,
      key: '2026-11-05',
    },
  ])('returns the $zoom of the date', ({ zoom, key }) => {
    expect(periodKey(date, zoom)).toBe(key);
  });

  it.each([
    {
      day: new Date(2027, 0, 1),
      key: '2026-W53',
    },
    {
      day: new Date(2025, 11, 29),
      key: '2026-W01',
    },
  ])('gives $day the ISO week of its Thursday', ({ day, key }) => {
    expect(periodKey(day, 'week')).toBe(key);
  });
});

describe('parentPeriod', () => {
  it.each([
    {
      period: '2026-11-05',
      parent: '2026-11',
    },
    {
      period: '2026-W45',
      parent: '2026',
    },
    {
      period: '2026-11',
      parent: '2026',
    },
    {
      period: '2026',
      parent: '2026',
    },
  ])('returns $parent for $period', ({ period, parent }) => {
    expect(parentPeriod(period)).toBe(parent);
  });
});

describe('periodStart', () => {
  it('returns the start of a day in the local time zone', () => {
    expect(periodStart('2026-11-05')).toEqual(new Date(2026, 10, 5));
  });

  it('returns the Monday of a week', () => {
    expect(periodStart('2026-W45')).toEqual(new Date(2026, 10, 2));
  });

  it('returns the Monday of a week that starts in the year before', () => {
    expect(periodStart('2026-W01')).toEqual(new Date(2025, 11, 29));
  });

  it('returns the start of a year', () => {
    expect(periodStart('2026')).toEqual(new Date(2026, 0, 1));
  });
});

describe('inPeriod', () => {
  it('reports a date inside the period', () => {
    expect(inPeriod(new Date(2026, 10, 8), '2026-W45', 'week')).toBe(true);
  });

  it('reports a date outside the period', () => {
    expect(inPeriod(new Date(2026, 10, 9), '2026-W45', 'week')).toBe(false);
  });
});

describe('timelineColumns', () => {
  it('fits more tiles into a row the smaller the zoom', () => {
    expect(timelineColumns(1320, 'year')).toBeGreaterThan(timelineColumns(1320, 'month'));
    expect(timelineColumns(1320, 'month')).toBeGreaterThan(timelineColumns(1320, 'week'));
    expect(timelineColumns(1320, 'week')).toBeGreaterThan(timelineColumns(1320, 'day'));
  });

  it('keeps one tile in a narrow width', () => {
    expect(timelineColumns(40, 'day')).toBe(1);
  });
});

describe('timelineRows', () => {
  const photos = [
    new Date(2026, 10, 20),
    new Date(2026, 10, 3),
    new Date(2026, 10, 1),
    new Date(2026, 9, 30),
  ];

  it('heads every period and breaks its photos into rows of the columns', () => {
    const rows = timelineRows(photos, (photo) => photo, 'month', 2);

    expect(rows.map((row) => [row.kind, row.items.length])).toEqual([
      ['header', 3],
      ['row', 2],
      ['row', 1],
      ['header', 1],
      ['row', 1],
    ]);
    expect(rows[0]).toMatchObject({
      period: '2026-11',
      start: photos[0],
    });
  });

  it('returns no rows without photos', () => {
    expect(timelineRows([], (photo: Date) => photo, 'day', 3)).toEqual([]);
  });
});

describe('periodOffset', () => {
  const periods = [
    {
      period: '2026-11',
      count: 3,
    },
    {
      period: '2026-09',
      count: 2,
    },
    {
      period: '2025-12',
      count: 4,
    },
  ];

  it('counts the photos before the period', () => {
    expect(periodOffset(periods, '2025-12')).toBe(5);
  });

  it('counts every photo for an unknown period', () => {
    expect(periodOffset(periods, '2024-01')).toBe(9);
  });
});
