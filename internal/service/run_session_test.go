package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/barthjs/photopi/internal/apitest"
)

func TestRunSessionService_Holds(t *testing.T) {
	t.Parallel()

	t.Run("does not hold another run", func(t *testing.T) {
		t.Parallel()

		sm := newTestSessionManager()
		svc := NewRunSessionService(sm)
		ctx := sessionContext(t, sm)
		run := apitest.Faker(t).IntRange(1, 1000)
		now := time.Now()
		svc.Start(ctx, run, now)

		held := svc.Holds(ctx, run+1, now)

		assert.False(t, held)
	})

	t.Run("does not hold an expired run", func(t *testing.T) {
		t.Parallel()

		sm := newTestSessionManager()
		svc := NewRunSessionService(sm)
		ctx := sessionContext(t, sm)
		run := apitest.Faker(t).IntRange(1, 1000)
		now := time.Now()
		svc.Start(ctx, run, now)

		held := svc.Holds(ctx, run, now.Add(runSessionTTL+time.Second))

		assert.False(t, held)
	})
}

func TestRunSessionService_Current(t *testing.T) {
	t.Parallel()

	t.Run("returns no run after its lifetime", func(t *testing.T) {
		t.Parallel()

		sm := newTestSessionManager()
		svc := NewRunSessionService(sm)
		ctx := sessionContext(t, sm)
		started := time.Now()
		svc.Start(ctx, apitest.Faker(t).IntRange(1, 1000), started)

		_, ok := svc.Current(ctx, started.Add(runSessionTTL+time.Second))

		assert.False(t, ok)
	})
}
