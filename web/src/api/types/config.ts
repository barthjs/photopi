export type AppSettings = {
  name: string;
  env: string;
  language: string;
  secret: string;
};

export type LogSettings = {
  level: string;
  channel: string;
  path: string;
  format: string;
  maxSizeMb: number;
  maxAgeDays: number;
  compress: boolean;
};

export type CameraSource = 'webcam' | 'rpicam' | 'gphoto2';

/** A fraction of a second like on a camera, or auto for a fully automatic exposure. */
export type CameraExposureTime =
  | 'auto'
  | '1/15'
  | '1/20'
  | '1/30'
  | '1/40'
  | '1/50'
  | '1/60'
  | '1/80'
  | '1/100'
  | '1/125'
  | '1/160'
  | '1/200'
  | '1/250'
  | '1/320'
  | '1/400'
  | '1/500';

export type CameraExposureMode = 'normal' | 'short';

export type CameraWhiteBalance =
  'auto' | 'incandescent' | 'tungsten' | 'fluorescent' | 'indoor' | 'daylight' | 'cloudy';

export type CameraRotation = 0 | 180;

export type CameraAutofocusMode = 'default' | 'manual' | 'auto' | 'continuous';

export type CameraDenoise = 'auto' | 'off' | 'cdn_off' | 'cdn_fast' | 'cdn_hq';

export type CameraSettings = {
  source: CameraSource;
  captureWidth: number;
  captureHeight: number;
  previewWidth: number;
  previewHeight: number;
  previewFps: number;
  overlayPath: string;
  exposureTime: CameraExposureTime;
  exposureMode: CameraExposureMode;
  analogueGain: number;
  exposureCompensation: number;
  whiteBalance: CameraWhiteBalance;
  horizontalFlip: boolean;
  verticalFlip: boolean;
  rotation: CameraRotation;
  jpegQuality: number;
  autofocusMode: CameraAutofocusMode;
  denoise: CameraDenoise;
};

export type CaptureSettings = {
  imagesPerRun: number;
  filePrefix: string;
  eventName: string;
  welcomeMessage: string;
  adminEmail: string;
  saveOriginals: boolean;
};

export type PrintSubmissionMode = 'raw' | 'managed';

export type PrintSettings = {
  enabled: boolean;
  printerName: string;
  layout: string;
  submissionMode: PrintSubmissionMode;
};

export type MailSettings = {
  driver: 'log' | 'smtp';
  host: string;
  port: number;
  username: string;
  password: string;
  fromAddress: string;
  fromName: string;
  replyTo: string;
  messageIdDomain: string;
};

export type ServerSettings = {
  host: string;
  port: number;
  trustedProxies: string;
  corsAllowedOrigins: string[];
  corsAllowCredentials: boolean;
  sslEnabled: boolean;
  sslCertFile: string;
  sslKeyFile: string;
};

export type EmailShareSettings = {
  enabled: boolean;
  subject: string;
};

export type NextcloudSettings = {
  url: string;
  username: string;
  password: string;
  folder: string;
};

export type SelfShareSettings = {
  baseUrl: string;
};

/** The empty string selects no cloud share driver. */
export type CloudShareDriver = '' | 'nextcloud' | 'self';

export type CloudShareSettings = {
  driver: CloudShareDriver;
  nextcloud: NextcloudSettings;
  self: SelfShareSettings;
};

export type ShareSettings = {
  email: EmailShareSettings;
  cloud: CloudShareSettings;
};

export type S3Settings = {
  endpoint: string;
  region: string;
  bucket: string;
  accessKey: string;
  secretKey: string;
  usePathStyle: boolean;
};

export type StorageSettings = {
  driver: 'local' | 's3';
  path: string;
  s3: S3Settings;
};

export type SystemSettings = {
  serverRestartCommand: string;
  serverStopCommand: string;
  kioskRestartCommand: string;
  kioskStopCommand: string;
  allRestartCommand: string;
  allStopCommand: string;
};

export type ThemeSettings = {
  primary: string;
  secondary: string;
  success: string;
  info: string;
  warning: string;
  error: string;
  neutral: string;
};

/** The config file a save writes to, with an empty path when the save creates a new one. */
export type ConfigFile = {
  path: string;
  exists: boolean;
};

/** Where the value of a setting comes from, with the name of the flag, the variable, or the file. */
export type ConfigSource = {
  kind: 'default' | 'file' | 'env' | 'flag';
  name: string;
};

/** Holds the sections that ConfigResponse and ConfigSaveRequest share. */
export type ConfigSettings = {
  app: AppSettings;
  log: LogSettings;
  camera: CameraSettings;
  capture: CaptureSettings;
  mail: MailSettings;
  print: PrintSettings;
  server: ServerSettings;
  share: ShareSettings;
  storage: StorageSettings;
  system: SystemSettings;
  theme: ThemeSettings;
};

/**
 * The configuration to save. A secret sent empty keeps its saved value,
 * unless its path is listed in clearSecrets.
 */
export type ConfigSaveRequest = ConfigSettings & {
  clearSecrets: string[];
};

/** The configuration without the values of its secrets, with the defaults and sources by path. */
export type ConfigResponse = ConfigSettings & {
  secrets: Record<string, boolean>;
  defaults: ConfigSettings;
  sources: Record<string, ConfigSource>;
  file: ConfigFile;
};

export type ConfigTestResponse = {
  ok: boolean;
  message?: string;
};
