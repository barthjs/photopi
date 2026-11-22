/** The size of the periods a timeline groups its photos by, from the smallest tiles to the largest. */
export type TimelineZoom = 'year' | 'month' | 'week' | 'day';

/** The zoom levels from the smallest tiles to the largest. */
export const TIMELINE_ZOOMS: readonly TimelineZoom[] = ['year', 'month', 'week', 'day'];

/** The smallest widths of a tile in pixels per zoom level. */
const TIMELINE_TILE_SIZES: Record<TimelineZoom, number> = {
  year: 72,
  month: 112,
  week: 160,
  day: 220,
};

/** A heading of a period or a row of photos below it. */
export type TimelineRow<T> =
  | {
      kind: 'header';
      key: string;
      // The period of the heading, see periodKey
      period: string;
      // The first photo of the period, which names it
      start: Date;
      // Every loaded photo of the period
      items: T[];
    }
  | {
      kind: 'row';
      key: string;
      items: T[];
    };

/** The heading of a period of a timeline. */
type TimelineHeader<T> = Extract<TimelineRow<T>, { kind: 'header' }>;

/** The number of photos in a period, given by its key, see periodKey. */
export type TimelinePeriod = {
  period: string;
  count: number;
};

/**
 * Returns the key of the period of the date at the zoom level in the local time zone,
 * as in 2026, 2026-11, 2026-W47, and 2026-11-17.
 */
export function periodKey(date: Date, zoom: TimelineZoom): string {
  switch (zoom) {
    case 'year':
      return String(date.getFullYear());
    case 'month':
      return `${date.getFullYear()}-${pad(date.getMonth() + 1)}`;
    case 'week': {
      const { year, week } = isoWeek(date);
      return `${year}-W${pad(week)}`;
    }
    case 'day':
      return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
  }
}

/** Writes a number of a date with at least two digits. */
function pad(value: number): string {
  return String(value).padStart(2, '0');
}

/** Returns the key of the period one zoom level larger that holds the period, the year for a year or a week. */
export function parentPeriod(period: string): string {
  // Only the key of a day is longer than that of a month, whose parent is the year like that of a week
  const isDay = period.length > 'YYYY-MM'.length && !period.includes('W');
  return isDay ? period.slice(0, 'YYYY-MM'.length) : period.slice(0, 'YYYY'.length);
}

/** Returns the start of the period of the key in the local time zone. */
export function periodStart(period: string): Date {
  const week = /^(\d{4})-W(\d{2})$/.exec(period);
  if (week) {
    const start = new Date(Number(week[1]), 0, 4);
    start.setDate(start.getDate() - weekday(start) + (Number(week[2]) - 1) * 7);
    return start;
  }
  const [year = 0, month = 1, day = 1] = period.split('-').map(Number);
  return new Date(year, month - 1, day);
}

/** Returns the ISO week of the date, which starts on Monday and belongs to the year of its Thursday. */
function isoWeek(date: Date): { year: number; week: number } {
  const thursday = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  thursday.setDate(thursday.getDate() + 3 - weekday(thursday));
  const firstThursday = new Date(thursday.getFullYear(), 0, 4);
  firstThursday.setDate(firstThursday.getDate() + 3 - weekday(firstThursday));
  // Rounding absorbs the hour a change of the daylight saving time adds or takes
  const weeks = Math.round(
    (thursday.getTime() - firstThursday.getTime()) / (7 * 24 * 60 * 60 * 1000),
  );
  return {
    year: thursday.getFullYear(),
    week: weeks + 1,
  };
}

/** Returns the day of the week of the date, counted from 0 on Monday. */
function weekday(date: Date): number {
  return (date.getDay() + 6) % 7;
}

/** Reports whether the date lies in the period of the key at the zoom level. */
export function inPeriod(date: Date, period: string, zoom: TimelineZoom): boolean {
  return periodKey(date, zoom) === period;
}

/** Returns the number of tiles per row that fit the width at the zoom level, at least one. */
export function timelineColumns(width: number, zoom: TimelineZoom): number {
  return Math.max(1, Math.floor(width / TIMELINE_TILE_SIZES[zoom]));
}

/**
 * Groups the photos, which come ordered by their date, into a heading per period
 * followed by rows of the number of columns.
 */
export function timelineRows<T>(
  items: T[],
  date: (item: T) => Date,
  zoom: TimelineZoom,
  columns: number,
): TimelineRow<T>[] {
  const rows: TimelineRow<T>[] = [];
  let header: TimelineHeader<T> | null = null;
  let row: TimelineRow<T> | null = null;
  for (const item of items) {
    const start = date(item);
    const period = periodKey(start, zoom);
    // eslint-disable-next-line @typescript-eslint/prefer-optional-chain -- an optional chain narrows header to never
    if (header === null || header.period !== period) {
      header = {
        kind: 'header',
        key: `header:${period}`,
        period,
        start,
        items: [],
      };
      rows.push(header);
      row = null;
    }
    header.items.push(item);
    if (!row || row.items.length === columns) {
      row = {
        kind: 'row',
        key: `row:${period}:${header.items.length}`,
        items: [],
      };
      rows.push(row);
    }
    row.items.push(item);
  }
  return rows;
}

/**
 * Returns the number of photos before the period in the order of the periods,
 * which a timeline loads before it can show the period.
 */
export function periodOffset(periods: TimelinePeriod[], period: string): number {
  let offset = 0;
  for (const entry of periods) {
    if (entry.period === period) return offset;
    offset += entry.count;
  }
  return offset;
}
