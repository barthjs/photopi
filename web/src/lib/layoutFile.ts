import type {
  LayoutBrandingResponse,
  LayoutDetailResponse,
  LayoutPageResponse,
  LayoutSlotResponse,
} from '@/api/types';

/**
 * parseLayout reads a layout from the JSON of a downloaded layout file. It only checks the shape the editor needs
 * to display it, the backend validates the remaining values on save. It returns null if the content is not a layout.
 */
export function parseLayout(json: string): LayoutDetailResponse | null {
  let value: unknown;
  try {
    value = JSON.parse(json);
  } catch {
    return null;
  }

  if (!isRecord(value) || !isRecord(value.paper) || !Array.isArray(value.pages)) return null;
  if (!isPositive(value.paper.widthMM) || !isPositive(value.paper.heightMM)) return null;

  const pages = parsePages(value.pages);
  const branding = parseBranding(value.branding);
  if (!pages || !branding) return null;

  return {
    id: typeof value.id === 'string' ? value.id : '',
    name: typeof value.name === 'string' ? value.name : '',
    paper: {
      widthMM: value.paper.widthMM,
      heightMM: value.paper.heightMM,
    },
    pages,
    branding,
  };
}

/** parsePages reads the pages of a layout file. It returns null if there are none or one of them is invalid. */
function parsePages(value: unknown[]): LayoutPageResponse[] | null {
  if (value.length === 0) return null;

  const pages: LayoutPageResponse[] = [];
  for (const page of value) {
    if (!isRecord(page) || !Array.isArray(page.slots)) return null;
    const slots = page.slots.map(parseSlot);
    if (slots.some((slot) => slot === null)) return null;
    pages.push({
      slots: slots.filter((slot) => slot !== null),
    });
  }
  return pages;
}

/** parseSlot reads one slot of a layout file, or returns null if it is invalid. */
function parseSlot(value: unknown): LayoutSlotResponse | null {
  if (!isRecord(value) || typeof value.locked !== 'boolean') return null;
  if (!isFraction(value.x) || !isFraction(value.y)) return null;
  if (!isFraction(value.w) || !isFraction(value.h) || value.w === 0 || value.h === 0) return null;

  return {
    x: value.x,
    y: value.y,
    w: value.w,
    h: value.h,
    locked: value.locked,
  };
}

/** parseBranding reads the caption of a layout file, or returns null if it is invalid. */
function parseBranding(value: unknown): LayoutBrandingResponse | null {
  if (
    !isRecord(value) ||
    typeof value.enabled !== 'boolean' ||
    typeof value.template !== 'string'
  ) {
    return null;
  }
  if (!isFraction(value.x) || !isFraction(value.y) || typeof value.rotationDeg !== 'number') {
    return null;
  }

  return {
    enabled: value.enabled,
    template: value.template,
    x: value.x,
    y: value.y,
    rotationDeg: value.rotationDeg,
    ...(isPositive(value.fontSizePt)
      ? {
          fontSizePt: value.fontSizePt,
        }
      : {}),
    ...(typeof value.color === 'string'
      ? {
          color: value.color,
        }
      : {}),
  };
}

/** isFraction reports whether a value is a number from 0 to 1, like positions and sizes on the page. */
function isFraction(value: unknown): value is number {
  return typeof value === 'number' && value >= 0 && value <= 1;
}

/** isPositive reports whether a value is a number above zero. */
function isPositive(value: unknown): value is number {
  return typeof value === 'number' && value > 0;
}

/** isRecord reports whether a value is a plain object. */
function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
