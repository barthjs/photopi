package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/barthjs/photopi/internal/apitest"
)

func TestRunJobs_begin(t *testing.T) {
	t.Parallel()

	t.Run("marks the job as pending", func(t *testing.T) {
		t.Parallel()

		jobs := newRunJobs()
		run := apitest.Faker(t).IntRange(1, 1000)

		started := jobs.begin(run)

		assert.True(t, started)
		state, found := jobs.get(run)
		assert.True(t, found)
		assert.Equal(t, JobState{
			Status: JobPending,
		}, state)
	})

	t.Run("starts a job again after the previous one finished", func(t *testing.T) {
		t.Parallel()

		jobs := newRunJobs()
		run := apitest.Faker(t).IntRange(1, 1000)
		jobs.begin(run)
		jobs.finish(run, JobState{
			Status: JobFailed,
			Reason: JobReasonFailed,
		})

		started := jobs.begin(run)

		assert.True(t, started)
		state, _ := jobs.get(run)
		assert.Equal(t, JobPending, state.Status)
	})

	t.Run("refuses a second job while one is pending", func(t *testing.T) {
		t.Parallel()

		jobs := newRunJobs()
		run := apitest.Faker(t).IntRange(1, 1000)
		jobs.begin(run)

		started := jobs.begin(run)

		assert.False(t, started)
	})

	t.Run("keeps the jobs of other runs apart", func(t *testing.T) {
		t.Parallel()

		jobs := newRunJobs()
		run := apitest.Faker(t).IntRange(1, 1000)
		jobs.begin(run)

		started := jobs.begin(run + 1)

		assert.True(t, started)
	})
}

func TestRunJobs_finish(t *testing.T) {
	t.Parallel()

	t.Run("records the outcome", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		jobs := newRunJobs()
		run := f.IntRange(1, 1000)
		outcome := JobState{
			Status: JobDone,
			URL:    f.URL(),
		}
		jobs.begin(run)

		jobs.finish(run, outcome)

		state, found := jobs.get(run)
		assert.True(t, found)
		assert.Equal(t, outcome, state)
	})
}

func TestRunJobs_get(t *testing.T) {
	t.Parallel()

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()

		jobs := newRunJobs()

		_, found := jobs.get(apitest.Faker(t).IntRange(1, 1000))

		assert.False(t, found)
	})
}
