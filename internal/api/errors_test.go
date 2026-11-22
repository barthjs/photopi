package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/service"
)

// TestAPI_RespondError checks the status and the code respondError answers an error with.
func TestAPI_RespondError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{
			name: "answers invalid settings as a bad request",
			err: &service.InvalidConfigError{
				Err: errors.New("invalid"),
			},
			status: http.StatusBadRequest,
			code:   codeBadRequest,
		},
		{
			name:   "answers an invalid sort field with a bad request",
			err:    apperr.ErrInvalidSort,
			status: http.StatusBadRequest,
			code:   codeBadRequest,
		},
		{
			name:   "answers a missing sign-in as unauthenticated",
			err:    apperr.ErrAuthenticationRequired,
			status: http.StatusUnauthorized,
			code:   codeAuthenticationRequired,
		},
		{
			name:   "answers a denied action as forbidden",
			err:    apperr.ErrForbidden,
			status: http.StatusForbidden,
			code:   codeForbidden,
		},
		{
			name:   "answers a concurrent modification with a conflict",
			err:    apperr.ErrVersionConflict,
			status: http.StatusConflict,
			code:   codeConflict,
		},
		{
			name:   "answers a busy run with a conflict",
			err:    service.ErrRunBusy,
			status: http.StatusConflict,
			code:   codeConflict,
		},
		{
			name:   "answers a run that is not open with a conflict",
			err:    service.ErrRunNotOpen,
			status: http.StatusConflict,
			code:   codeConflict,
		},
		{
			name:   "answers an active overlay with a conflict",
			err:    service.ErrOverlayActive,
			status: http.StatusConflict,
			code:   codeConflict,
		},
		{
			name:   "answers an unavailable print queue with a conflict",
			err:    service.ErrPrintQueueUnavailable,
			status: http.StatusConflict,
			code:   codeConflict,
		},
		{
			name:   "answers a corrupt capture as a failed camera capture",
			err:    service.ErrCaptureCorrupt,
			status: http.StatusBadGateway,
			code:   codeCameraCaptureFailed,
		},
		{
			name:   "answers any other error with a server error",
			err:    errors.New("unknown"),
			status: http.StatusInternalServerError,
			code:   codeServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			err := fmt.Errorf("%s: %w", apitest.Faker(t).Word(), tt.err)

			newBareAPI().respondError(recorder, request, err)

			assert.Equal(t, tt.status, recorder.Code)
			assert.Equal(t, tt.code, errorCode(t, recorder))
		})
	}
}

func TestAPI_ServerErrorResponse(t *testing.T) {
	t.Parallel()

	t.Run("answers with a server error", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		err := errors.New(apitest.Faker(t).Word())

		newBareAPI().serverErrorResponse(recorder, request, err)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, codeServerError, errorCode(t, recorder))
	})

	t.Run("answers a timed out request with a timeout", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		err := fmt.Errorf("%s: %w", apitest.Faker(t).Word(), context.DeadlineExceeded)

		newBareAPI().serverErrorResponse(recorder, request, err)

		assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
		assert.Equal(t, codeTimeout, errorCode(t, recorder))
	})
}
