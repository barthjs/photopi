package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
)

func TestRunSessionService_Owner(t *testing.T) {
	t.Parallel()

	t.Run("creates a random owner", func(t *testing.T) {
		t.Parallel()

		sm := newTestSessionManager()
		svc := NewRunSessionService(sm)

		owner, err := svc.Owner(sessionContext(t, sm))

		require.NoError(t, err)
		assert.Len(t, owner, runOwnerLength)
	})

	t.Run("keeps the owner of the session", func(t *testing.T) {
		t.Parallel()

		sm := newTestSessionManager()
		svc := NewRunSessionService(sm)
		ctx := sessionContext(t, sm)
		first, err := svc.Owner(ctx)
		require.NoError(t, err)

		second, err := svc.Owner(ctx)

		require.NoError(t, err)
		assert.Equal(t, first, second)
	})

	t.Run("gives every session its own owner", func(t *testing.T) {
		t.Parallel()

		sm := newTestSessionManager()
		svc := NewRunSessionService(sm)
		first, err := svc.Owner(sessionContext(t, sm))
		require.NoError(t, err)

		second, err := svc.Owner(sessionContext(t, sm))

		require.NoError(t, err)
		assert.NotEqual(t, first, second)
	})
}

func TestRunSessionService_Start(t *testing.T) {
	t.Parallel()

	t.Run("binds the run to the session", func(t *testing.T) {
		t.Parallel()

		sm := newTestSessionManager()
		svc := NewRunSessionService(sm)
		ctx := sessionContext(t, sm)
		run := apitest.Faker(t).IntRange(1, 1000)
		now := time.Now()

		svc.Start(ctx, run, now)

		current, ok := svc.Current(ctx, now)
		assert.True(t, ok)
		assert.Equal(t, run, current)
	})

	t.Run("replaces the previous run", func(t *testing.T) {
		t.Parallel()

		sm := newTestSessionManager()
		svc := NewRunSessionService(sm)
		ctx := sessionContext(t, sm)
		run := apitest.Faker(t).IntRange(1, 1000)
		now := time.Now()
		svc.Start(ctx, run, now)

		svc.Start(ctx, run+1, now)

		current, ok := svc.Current(ctx, now)
		assert.True(t, ok)
		assert.Equal(t, run+1, current)
	})
}

func TestRunSessionService_Holds(t *testing.T) {
	t.Parallel()

	t.Run("holds the run of the session", func(t *testing.T) {
		t.Parallel()

		sm := newTestSessionManager()
		svc := NewRunSessionService(sm)
		ctx := sessionContext(t, sm)
		run := apitest.Faker(t).IntRange(1, 1000)
		now := time.Now()
		svc.Start(ctx, run, now)

		held := svc.Holds(ctx, run, now)

		assert.True(t, held)
	})

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

	t.Run("returns the run of the session within its lifetime", func(t *testing.T) {
		t.Parallel()

		sm := newTestSessionManager()
		svc := NewRunSessionService(sm)
		ctx := sessionContext(t, sm)
		run := apitest.Faker(t).IntRange(1, 1000)
		// The session keeps the start in whole seconds
		started := time.Unix(time.Now().Unix(), 0)
		svc.Start(ctx, run, started)

		current, ok := svc.Current(ctx, started.Add(runSessionTTL))

		assert.True(t, ok)
		assert.Equal(t, run, current)
	})

	t.Run("returns no run without a started one", func(t *testing.T) {
		t.Parallel()

		sm := newTestSessionManager()
		svc := NewRunSessionService(sm)

		_, ok := svc.Current(sessionContext(t, sm), time.Now())

		assert.False(t, ok)
	})

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
