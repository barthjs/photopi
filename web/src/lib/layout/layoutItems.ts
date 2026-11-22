import type { LayoutResponse } from '@/api/types/layout';

/** A layout as an item of a select menu, which shows the description below the name. */
export type LayoutMenuItem = Omit<LayoutResponse, 'description'> & {
  description?: string;
};

/** Returns the layouts as items of a select menu, which leaves out a missing description. */
export function layoutMenuItems(layouts: LayoutResponse[]): LayoutMenuItem[] {
  return layouts.map(({ description, ...layout }) => ({
    ...layout,
    ...(description
      ? {
          description,
        }
      : {}),
  }));
}
