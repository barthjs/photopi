/** Narrows the runs of the gallery to one event. */
export type GalleryListFilters = {
  event?: string;
};

/** An event with stored runs, current marks the event the booth files new captures under. */
export type GalleryEventResponse = {
  event: string;
  current: boolean;
  imageCount: number;
  originalCount: number;
};

export type GalleryRunResponse = {
  run: number;
  event: string;
  imageCount: number;
  // The number of captures stored without the overlay as well
  originalCount: number;
  // The time of the first photo as the wall time of the booth without a time zone, null while it is unknown
  takenAt: string | null;
};

/** The number of runs taken on a day, given as YYYY-MM-DD. */
export type GalleryDayResponse = {
  day: string;
  count: number;
};

/** Holds the number of runs per day, the newest day first. */
export type GalleryDayListResponse = {
  days: GalleryDayResponse[];
};
