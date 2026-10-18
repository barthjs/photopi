//go:build unix

package camera_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// frame is the JPEG every fake preview program prints, as octal escapes for printf.
const frame = `\377\330preview\377\331`

// still is what every fake still program prints.
const still = "\xff\xd8still\xff\xd9"

// fakeTools puts shell scripts named like the camera tools first on PATH, so the backends run them
// instead of the real programs. Every script records its arguments in <name>.args and the libcamera log levels
// of its environment in <name>.env. A file named <name>.fail makes it exit with an error, <name>.warn makes it write a warning to stderr,
// and <name>.info a progress message.
type fakeTools struct {
	dir   string
	sleep string
}

// newFakeTools replaces PATH with a directory of fake tools only, so a camera tool installed on the machine
// can never run. The scripts therefore use shell builtins and the absolute path of sleep.
func newFakeTools(t *testing.T) *fakeTools {
	t.Helper()

	sleep, err := exec.LookPath("sleep")
	require.NoError(t, err)

	dir := t.TempDir()
	t.Setenv("PATH", dir)
	return &fakeTools{
		dir:   dir,
		sleep: sleep,
	}
}

// preview writes a fake live view program that prints count frames, or frames until it is stopped for count 0.
// It exits on SIGTERM like the real tools.
func (f *fakeTools) preview(t *testing.T, name string, count int) {
	t.Helper()

	loop := "while true; do printf '" + frame + "'; " + f.sleep + " 0.02; done"
	if count > 0 {
		loop = "i=0; while [ $i -lt " + strconv.Itoa(count) + " ]; do printf '" + frame + "'; i=$((i+1)); done"
	}
	f.script(t, name, "trap 'exit 0' TERM\n"+loop+"\n")
}

// hangingStill writes a fake still program that never prints a still and exits on SIGTERM like the real tools.
func (f *fakeTools) hangingStill(t *testing.T, name string) {
	t.Helper()
	f.script(t, name, "trap 'kill $!; exit 0' TERM\n"+f.sleep+" 30 >/dev/null 2>&1 &\nwait $!\n")
}

// still writes a fake still program that prints still.
func (f *fakeTools) still(t *testing.T, name string) {
	t.Helper()
	f.script(t, name, `printf '\377\330still\377\331'`+"\n")
}

// script writes a fake program whose body runs after the shared argument and failure handling.
func (f *fakeTools) script(t *testing.T, name, body string) {
	t.Helper()

	header := `#!/bin/sh
dir=${0%/*}
echo "$@" > "$dir/` + name + `.args"
echo "$LIBCAMERA_LOG_LEVELS" > "$dir/` + name + `.env"
if [ -f "$dir/` + name + `.fail" ]; then echo "device busy" >&2; exit 1; fi
if [ -f "$dir/` + name + `.warn" ]; then echo "WARN low light" >&2; fi
if [ -f "$dir/` + name + `.info" ]; then echo "camera started" >&2; fi
`
	require.NoError(t, os.WriteFile(filepath.Join(f.dir, name), []byte(header+body), 0o755))
}

// flag creates a marker file such as <name>.fail.
func (f *fakeTools) flag(t *testing.T, name string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(f.dir, name), nil, 0o644))
}

// args returns the arguments the program was last started with.
func (f *fakeTools) args(t *testing.T, name string) string {
	t.Helper()
	return f.read(t, name+".args")
}

// env returns the libcamera log levels the program was last started with.
func (f *fakeTools) env(t *testing.T, name string) string {
	t.Helper()
	return f.read(t, name+".env")
}

// read returns the trimmed content of a file the programs wrote.
func (f *fakeTools) read(t *testing.T, file string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(f.dir, file))
	require.NoError(t, err)
	return strings.TrimSpace(string(data))
}

// syncBuffer is a bytes.Buffer safe for the concurrent writes of a running preview and a reading test.
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

// newLogger returns a logger writing JSON lines to the returned buffer.
func newLogger() (zerolog.Logger, *syncBuffer) {
	buf := &syncBuffer{}
	return zerolog.New(buf), buf
}
