//go:build unix

package service

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
)

func TestExecRunner_Start(t *testing.T) {
	t.Parallel()

	t.Run("starts without waiting", func(t *testing.T) {
		t.Parallel()

		runner := NewExecRunner(zerolog.Nop())

		err := runner.Start("true")

		require.NoError(t, err)
	})

	t.Run("logs a program that fails later", func(t *testing.T) {
		t.Parallel()

		var logs syncBuffer
		message := apitest.Faker(t).LetterN(12)
		runner := NewExecRunner(zerolog.New(&logs))

		err := runner.Start("sh", "-c", "echo "+message+"; exit 2")

		require.NoError(t, err)
		assert.Eventually(t, func() bool { return strings.Contains(logs.String(), message) }, 5*time.Second, 10*time.Millisecond)
	})

	t.Run("rejects an unknown program", func(t *testing.T) {
		t.Parallel()

		runner := NewExecRunner(zerolog.Nop())

		err := runner.Start("photopi-command-that-does-not-exist")

		assert.Error(t, err)
	})
}

func TestExecRunner_Output(t *testing.T) {
	t.Parallel()

	t.Run("returns the output with extra environment", func(t *testing.T) {
		t.Parallel()

		var runner CommandRunner = NewExecRunner(zerolog.Nop())
		value := apitest.Faker(t).LetterN(12)

		out, err := runner.Output(t.Context(), []string{"PHOTOPI_TEST=" + value},
			"sh", "-c", `printf "%s" "$PHOTOPI_TEST"`)

		require.NoError(t, err)
		assert.Equal(t, value, string(out))
	})

	t.Run("keeps the process environment", func(t *testing.T) {
		t.Parallel()

		var runner CommandRunner = NewExecRunner(zerolog.Nop())

		out, err := runner.Output(t.Context(), nil, "sh", "-c", `printf "%s" "$PATH"`)

		require.NoError(t, err)
		assert.NotEmpty(t, strings.TrimSpace(string(out)))
	})

	t.Run("returns the output and an exit error for a non-zero exit", func(t *testing.T) {
		t.Parallel()

		var runner CommandRunner = NewExecRunner(zerolog.Nop())

		out, err := runner.Output(t.Context(), nil, "sh", "-c", "echo partial; exit 3")

		_, isExit := errors.AsType[*exec.ExitError](err)
		assert.True(t, isExit, "%v", err)
		assert.Equal(t, "partial\n", string(out))
	})
}

func TestExecRunner_CombinedOutput(t *testing.T) {
	t.Parallel()

	t.Run("includes standard error", func(t *testing.T) {
		t.Parallel()

		var runner CommandRunner = NewExecRunner(zerolog.Nop())

		out, err := runner.CombinedOutput(t.Context(), nil, "sh", "-c", "echo out; echo err >&2")

		require.NoError(t, err)
		assert.Equal(t, "out\nerr\n", string(out))
	})
}

// syncBuffer is a bytes.Buffer that is safe for a writer and a reader in different goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write appends p to the buffer.
func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// String returns the content written so far.
func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
