/** Holds the configured shell commands of the system actions, an empty one is not configured. */
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
