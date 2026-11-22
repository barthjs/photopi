export type LayoutSlotRequest = {
  x: number;
  y: number;
  w: number;
  h: number;
  // Keeps the slot in place in the editor, printing ignores it
  locked: boolean;
};

/**
 * Holds the slots of one printed page. The captures of a run fill the pages in order,
 * so a run with more photos than slots on one page continues on the next.
 */
export type LayoutPageRequest = {
  slots: LayoutSlotRequest[];
};

export type LayoutPaperRequest = {
  widthMM: number;
  heightMM: number;
};

export type LayoutBrandingRequest = {
  enabled: boolean;
  template: string;
  // The center of the caption as a fraction of the page, like the position of a slot
  x: number;
  y: number;
  rotationDeg: number;
  // Left unset, the server prints with its default size and color
  fontSizePt?: number;
  color?: string;
};

export type LayoutSaveRequest = {
  name: string;
  description: string | null;
  paper: LayoutPaperRequest;
  pages: LayoutPageRequest[];
  branding: LayoutBrandingRequest;
};

export type LayoutSlotResponse = {
  x: number;
  y: number;
  w: number;
  h: number;
  // Keeps the slot in place in the editor, printing ignores it
  locked: boolean;
};

export type LayoutPageResponse = {
  slots: LayoutSlotResponse[];
};

export type LayoutPaperResponse = {
  widthMM: number;
  heightMM: number;
};

export type LayoutBrandingResponse = {
  enabled: boolean;
  template: string;
  x: number;
  y: number;
  rotationDeg: number;
  // Missing if the layout prints with the default size and color
  fontSizePt?: number;
  color?: string;
};

export type LayoutResponse = {
  name: string;
  description: string | null;
  paperWidthMM: number;
  paperHeightMM: number;
  // The number of slots on all pages together
  slots: number;
  pages: number;
  // The layout the print settings use, which can be neither deleted nor renamed
  inUse: boolean;
};

/** One layout with its pages and slots, as the editor loads and saves it. */
export type LayoutDetailResponse = {
  name: string;
  description: string | null;
  paper: LayoutPaperResponse;
  pages: LayoutPageResponse[];
  branding: LayoutBrandingResponse;
};
