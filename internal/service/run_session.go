package service

import (
	"context"
	"fmt"
	"time"

	"github.com/alexedwards/scs/v2"

	"github.com/barthjs/photopi/internal/crypto"
)

const (
	// sessionKeyRunOwner identifies the client of a session towards the CaptureService.
	sessionKeyRunOwner = "runOwner"
	// sessionKeyRun is the run the session started last.
	sessionKeyRun = "run"
	// sessionKeyRunStarted is when the session started that run, in Unix seconds.
	sessionKeyRunStarted = "runStarted"

	// runSessionTTL is how long a session may access the run it started. It covers the captures, the review,
	// and the share screen of one guest, so a forgotten run is not reachable for long.
	runSessionTTL = 30 * time.Minute
	// runOwnerLength is the length of the random owner of a session.
	runOwnerLength = 24
)

// RunSessionService binds a run to the session that started it,
// so only the client of the guest reaches its captures, its shares, and the live preview.
type RunSessionService struct {
	sm *scs.SessionManager
}

// NewRunSessionService creates a RunSessionService.
func NewRunSessionService(sm *scs.SessionManager) *RunSessionService {
	return &RunSessionService{
		sm: sm,
	}
}

// Owner returns the random owner of the current session, creating it on first use.
func (s *RunSessionService) Owner(ctx context.Context) (string, error) {
	if owner := s.sm.GetString(ctx, sessionKeyRunOwner); owner != "" {
		return owner, nil
	}

	owner, err := crypto.GenerateRandomCode(runOwnerLength)
	if err != nil {
		return "", fmt.Errorf("service: generating run owner: %w", err)
	}
	s.sm.Put(ctx, sessionKeyRunOwner, owner)

	return owner, nil
}

// Start binds run to the current session, replacing the run it started before.
func (s *RunSessionService) Start(ctx context.Context, run int, now time.Time) {
	s.sm.Put(ctx, sessionKeyRun, run)
	s.sm.Put(ctx, sessionKeyRunStarted, now.Unix())
}

// Holds reports whether the current session started run within runSessionTTL.
func (s *RunSessionService) Holds(ctx context.Context, run int, now time.Time) bool {
	held, ok := s.Current(ctx, now)
	return ok && held == run
}

// Current returns the run the current session started within runSessionTTL, ok is false if there is none.
func (s *RunSessionService) Current(ctx context.Context, now time.Time) (run int, ok bool) {
	run = s.sm.GetInt(ctx, sessionKeyRun)
	if run == 0 {
		return 0, false
	}

	started := time.Unix(s.sm.GetInt64(ctx, sessionKeyRunStarted), 0)
	if now.Sub(started) > runSessionTTL {
		return 0, false
	}

	return run, true
}
