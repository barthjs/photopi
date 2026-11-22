import type { CameraSource, ThemeSettings } from '@/api/types/config';

export type CameraMeta = {
  source: CameraSource;
  hasOverlay: boolean;
};

export type CaptureMeta = {
  imagesPerRun: number;
  welcomeMessage: string;
};

export type ShareMeta = {
  emailEnabled: boolean;
  cloudEnabled: boolean;
  printEnabled: boolean;
  adminEmail: string;
};

export type MetaResponse = {
  name: string;
  version: string;
  // Changes with every start of the backend
  instance: string;
  language: string;
  camera: CameraMeta;
  capture: CaptureMeta;
  share: ShareMeta;
  theme: ThemeSettings;
};
