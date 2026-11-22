export type ShareJobStatus = 'pending' | 'done' | 'failed';

/** Names why a background share job failed. */
export type ShareJobReason = 'unreachable' | 'queue_unavailable' | 'failed';

export type ShareEmailRequest = {
  email: string;
};

/** The state of a background share job of a run, with the link of a finished cloud upload. */
export type ShareJobResponse = {
  status: ShareJobStatus;
  reason?: ShareJobReason;
  url?: string;
};
