export type AuthLoginRequest = {
  password: string;
};

export type AuthStatusResponse = {
  authenticated: boolean;
};

export type CaptureResponse = {
  run: number;
  imageIndex: number;
  imagesPerRun: number;
};

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

/** CameraExposureTime is a fraction of a second like on a camera, or auto for a fully automatic exposure. */
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
  layoutId: string;
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

/**
 * ConfigFile is the config file a save writes to. path is empty if none was found at startup,
 * a save then creates one. exists is false until the file at path is created.
 */
export type ConfigFile = {
  path: string;
  exists: boolean;
};

/**
 * ConfigSource is where the value of a setting comes from. A flag or an environment variable
 * overrides the config file. name is the flag, the variable, or the file path, empty for the default.
 */
export type ConfigSource = {
  kind: 'default' | 'file' | 'env' | 'flag';
  name: string;
};

/** ConfigSettings holds the sections that ConfigResponse and ConfigSaveRequest share. */
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
 * ConfigSaveRequest is the configuration to save. A secret sent empty keeps its saved value,
 * unless its path is listed in clearSecrets.
 */
export type ConfigSaveRequest = ConfigSettings & {
  clearSecrets: string[];
};

/**
 * ConfigResponse is the configuration without its secrets. secrets reports by path whether each secret is set,
 * since the server never sends their values. defaults holds the default of every setting,
 * sources reports by path where each setting comes from.
 */
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

/** GalleryListFilters narrows the runs of the gallery to one event. */
export type GalleryListFilters = {
  event?: string;
};

/** GalleryEventResponse is an event with stored runs, current marks the event the booth files new captures under. */
export type GalleryEventResponse = {
  event: string;
  current: boolean;
  runCount: number;
  imageCount: number;
  originalCount: number;
};

export type GalleryRunResponse = {
  run: number;
  event: string;
  imageCount: number;
  // The number of captures stored without the overlay as well.
  originalCount: number;
};

export type LayoutSlotRequest = {
  x: number;
  y: number;
  w: number;
  h: number;
  // Keeps the slot in place in the editor, printing ignores it.
  locked: boolean;
};

/**
 * LayoutPageRequest holds the slots of one printed page. The captures of a run fill the pages in order,
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
  // The center of the caption as a fraction of the page, like the position of a slot.
  x: number;
  y: number;
  rotationDeg: number;
  // Left unset, the server prints with its default size and color.
  fontSizePt?: number;
  color?: string;
};

export type LayoutSaveRequest = {
  name: string;
  paper: LayoutPaperRequest;
  pages: LayoutPageRequest[];
  branding: LayoutBrandingRequest;
};

export type LayoutSlotResponse = {
  x: number;
  y: number;
  w: number;
  h: number;
  // Keeps the slot in place in the editor, printing ignores it.
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
  // Missing if the layout prints with the default size and color.
  fontSizePt?: number;
  color?: string;
};

export type LayoutResponse = {
  id: string;
  name: string;
  paperWidthMM: number;
  paperHeightMM: number;
  // The number of slots on all pages together.
  slots: number;
  pages: number;
};

/** LayoutDetailResponse is one layout with its pages and slots, as the editor loads and saves it. */
export type LayoutDetailResponse = {
  id: string;
  name: string;
  paper: LayoutPaperResponse;
  pages: LayoutPageResponse[];
  branding: LayoutBrandingResponse;
};

export type CameraMeta = {
  source: CameraSource;
  captureWidth: number;
  captureHeight: number;
  previewWidth: number;
  previewHeight: number;
  previewFps: number;
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
  // Changes with every start of the backend.
  instance: string;
  language: string;
  camera: CameraMeta;
  capture: CaptureMeta;
  share: ShareMeta;
  theme: ThemeSettings;
};

export type OverlayResponse = {
  name: string;
  path: string;
};

export type PrintCancelRequest = {
  printerName: string;
};

export type PrintResumeRequest = {
  printerName: string;
};

export type PrintTestRequest = {
  printerName: string;
};

export type PrinterResponse = {
  name: string;
  status: string;
};

export type PublicShareResponse = {
  // Signed URLs of the captures followed by the collage pages, valid for a limited time.
  imageUrls: string[];
};

export type RunResponse = {
  run: number;
  imagesPerRun: number;
};

export type ShareJobStatus = 'pending' | 'done' | 'failed';

/** ShareJobReason names why a background share job failed. */
export type ShareJobReason = 'unreachable' | 'queue_unavailable' | 'failed';

export type ShareEmailRequest = {
  email: string;
};

/**
 * ShareJobResponse is the state of a background share job of a run, such as an email, a cloud upload,
 * or a print. url is the link of a finished cloud upload.
 */
export type ShareJobResponse = {
  status: ShareJobStatus;
  reason?: ShareJobReason;
  url?: string;
};

/** SystemCommandsResponse holds the configured shell commands of the system actions, an empty one is not configured. */
export type SystemCommandsResponse = {
  serverRestart: string;
  serverStop: string;
  kioskRestart: string;
  kioskStop: string;
  allRestart: string;
  allStop: string;
};

export type SystemInfoResponse = {
  version: string;
  uptimeSeconds: number;
  hostname: string;
  ips: string[];
  cpuPercent: number;
  memUsed: number;
  memTotal: number;
  diskUsed: number;
  diskTotal: number;
  commands: SystemCommandsResponse;
};

/**
 * UpdateCheckResponse compares the running version with the latest release.
 * unsupported names why this installation cannot install it itself, missing if it can.
 */
export type UpdateCheckResponse = {
  current: string;
  latest: string;
  url: string;
  newer: boolean;
  unsupported?: 'development_build' | 'container' | 'package_managed' | 'not_writable';
};

export type UpdateStatusResponse = {
  state: 'idle' | 'running' | 'failed' | 'applied';
  version?: string;
  error?: string;
};
