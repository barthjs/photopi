export const cameraRepository = {
  /** previewUrl returns the address of the live preview stream. */
  previewUrl(): string {
    return '/api/camera/preview';
  },

  /** overlayUrl returns the address of the overlay drawn over the preview. */
  overlayUrl(): string {
    return '/api/camera/overlay';
  },
};
