package service

import (
	"sync"
)

// JobStatus is the state of a background job of a run, such as an email or a cloud upload.
type JobStatus string

const (
	JobPending JobStatus = "pending"
	JobDone    JobStatus = "done"
	JobFailed  JobStatus = "failed"
)

// JobReason names why a background job failed, so the guest gets a message that fits.
type JobReason string

const (
	// JobReasonNone is the reason of a job that did not fail.
	JobReasonNone JobReason = ""
	// JobReasonUnreachable means the destination could not be reached, for example without internet.
	JobReasonUnreachable JobReason = "unreachable"
	// JobReasonQueueUnavailable means the printer queue did not accept the print,
	// for example because it is stopped or already holds too many jobs.
	JobReasonQueueUnavailable JobReason = "queue_unavailable"
	// JobReasonFailed covers every other failure.
	JobReasonFailed JobReason = "failed"
)

// JobState is the recorded state of the latest background job of one kind for a run.
type JobState struct {
	Status JobStatus
	Reason JobReason
	// URL is the share link of a finished cloud upload, empty for every other job.
	URL string
}

// runJobs records the state of the latest background job of one kind per run, for polling.
// A run has at most one job of a kind in flight at a time.
type runJobs struct {
	mu   sync.Mutex
	jobs map[int]JobState
}

// newRunJobs creates an empty runJobs.
func newRunJobs() *runJobs {
	return &runJobs{
		jobs: make(map[int]JobState),
	}
}

// begin marks the job of run as pending and reports whether the caller should start it,
// which is false while a job of the run is already pending.
func (j *runJobs) begin(run int) bool {
	j.mu.Lock()
	defer j.mu.Unlock()

	if existing, ok := j.jobs[run]; ok && existing.Status == JobPending {
		return false
	}

	j.jobs[run] = JobState{
		Status: JobPending,
	}
	return true
}

// finish records the outcome of the job of run.
func (j *runJobs) finish(run int, state JobState) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.jobs[run] = state
}

// get returns the state of the job of run, found is false if no job was ever started for it.
func (j *runJobs) get(run int) (state JobState, found bool) {
	j.mu.Lock()
	defer j.mu.Unlock()

	state, found = j.jobs[run]
	return state, found
}
