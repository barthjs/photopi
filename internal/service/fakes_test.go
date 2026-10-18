package service

import (
	"context"
	"os"
	"sync"
)

// commandCall is one recorded call of fakeCommands.
type commandCall struct {
	Method string
	Name   string
	Args   []string
	Env    []string
	// File holds the content of the last argument if it named a readable file at the time of the call,
	// since the print service deletes its temporary files right after submitting them.
	File []byte
}

// fakeCommands records calls instead of running programs and answers with scripted results.
type fakeCommands struct {
	mu    sync.Mutex
	calls []commandCall

	startErr error
	// respond answers Output and CombinedOutput. Nil answers with empty output.
	respond func(call commandCall) ([]byte, error)
}

// Start records the call and returns startErr.
func (f *fakeCommands) Start(name string, args ...string) error {
	f.record(commandCall{
		Method: "Start",
		Name:   name,
		Args:   args,
	})
	return f.startErr
}

// Output records the call and answers through respond.
func (f *fakeCommands) Output(_ context.Context, env []string, name string, args ...string) ([]byte, error) {
	return f.answer(f.record(commandCall{
		Method: "Output",
		Name:   name,
		Args:   args,
		Env:    env,
	}))
}

// CombinedOutput records the call and answers through respond.
func (f *fakeCommands) CombinedOutput(_ context.Context, env []string, name string, args ...string) ([]byte, error) {
	return f.answer(f.record(commandCall{
		Method: "CombinedOutput",
		Name:   name,
		Args:   args,
		Env:    env,
	}))
}

// answer returns the scripted result of a call.
func (f *fakeCommands) answer(call commandCall) ([]byte, error) {
	if f.respond == nil {
		return nil, nil
	}
	return f.respond(call)
}

// record stores the call together with the file its last argument names.
func (f *fakeCommands) record(call commandCall) commandCall {
	if len(call.Args) > 0 {
		if data, err := os.ReadFile(call.Args[len(call.Args)-1]); err == nil {
			call.File = data
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call)
	return call
}

// Calls returns a copy of the recorded calls.
func (f *fakeCommands) Calls() []commandCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]commandCall(nil), f.calls...)
}
