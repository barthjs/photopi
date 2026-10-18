import { describe, expect, it } from 'vitest';

import { cameraRepository } from '@/api/repositories/camera';

describe('cameraRepository', () => {
  describe('previewUrl', () => {
    it('returns the address of the live preview', () => {
      expect(cameraRepository.previewUrl()).toBe('/api/camera/preview');
    });
  });

  describe('overlayUrl', () => {
    it('returns the address of the overlay', () => {
      expect(cameraRepository.overlayUrl()).toBe('/api/camera/overlay');
    });
  });
});
