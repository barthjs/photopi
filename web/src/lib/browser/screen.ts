import type { RunCreateRequest } from '@/api/types/run';

/** The part of the window that tells the size of the screen. */
type ScreenView = Pick<Window, 'innerWidth' | 'innerHeight' | 'devicePixelRatio'>;

/** Returns the size of the view in device pixels, so the backend sizes images for a sharp display. */
export function screenPixels(view: ScreenView): RunCreateRequest {
  const ratio = view.devicePixelRatio || 1;
  return {
    screenWidth: Math.max(1, Math.round(view.innerWidth * ratio)),
    screenHeight: Math.max(1, Math.round(view.innerHeight * ratio)),
  };
}
