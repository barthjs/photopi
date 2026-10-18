package service

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog"
)

// CommandRunner runs external programs. ExecRunner implements it,
// and tests replace it, since the programs act on the printer and the host.
type CommandRunner interface {
	// Start starts a program without waiting for it to finish. It only reports whether starting failed.
	// Whether the program went on to succeed is logged once it exits.
	Start(name string, args ...string) error
	// Output runs a program with env appended to the process environment and returns its standard output.
	// A program that ran but exited non-zero returns its output together with an *exec.ExitError.
	Output(ctx context.Context, env []string, name string, args ...string) ([]byte, error)
	// CombinedOutput runs a program like Output, but returns standard output and error together.
	CombinedOutput(ctx context.Context, env []string, name string, args ...string) ([]byte, error)
}

// ExecRunner runs programs on the host through os/exec.
type ExecRunner struct {
	logger zerolog.Logger
}

// NewExecRunner creates an ExecRunner that logs the failures of started programs to logger.
func NewExecRunner(logger zerolog.Logger) ExecRunner {
	return ExecRunner{
		logger: logger,
	}
}

// Start starts the program without waiting for it to finish. A goroutine reaps the exited program
// and logs a failure with its output, since nobody else learns about it.
func (r ExecRunner) Start(name string, args ...string) error {
	var output bytes.Buffer
	cmd := exec.Command(name, args...) //nolint:noctx // The program may outlive the request that started it, so no context ends it.
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			r.logger.Warn().
				Err(err).
				Str("command", name).
				Strs("args", args).
				Str("output", strings.TrimSpace(output.String())).
				Msg("started command failed")
		}
	}()

	return nil
}

// Output runs the program and returns its standard output.
func (ExecRunner) Output(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	return command(ctx, env, name, args).Output()
}

// CombinedOutput runs the program and returns its standard output and error together.
func (ExecRunner) CombinedOutput(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	return command(ctx, env, name, args).CombinedOutput()
}

// command builds the command for a program with env added to the environment of the process.
func command(ctx context.Context, env []string, name string, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	return cmd
}
