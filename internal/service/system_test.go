package service

import (
	"net"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRebootCommand(t *testing.T) {
	t.Parallel()

	t.Run("returns the reboot command of the platform", func(t *testing.T) {
		t.Parallel()

		name, args := rebootCommand()

		if runtime.GOOS == "windows" {
			assert.Equal(t, "shutdown", name)
			assert.Equal(t, []string{"/r", "/t", "0"}, args)
			return
		}
		assert.Equal(t, "reboot", name)
		assert.Empty(t, args)
	})
}

func TestShutdownCommand(t *testing.T) {
	t.Parallel()

	t.Run("returns the shutdown command of the platform", func(t *testing.T) {
		t.Parallel()

		name, args := shutdownCommand()

		if runtime.GOOS == "windows" {
			assert.Equal(t, []string{"/s", "/t", "0"}, args)
			return
		}
		assert.Equal(t, "shutdown", name)
		assert.Equal(t, []string{"-h", "now"}, args)
	})
}

func TestDiskUsagePath(t *testing.T) {
	t.Parallel()

	t.Run("returns the root of the platform", func(t *testing.T) {
		t.Parallel()

		path := diskUsagePath()

		if runtime.GOOS == "windows" {
			assert.Equal(t, `C:\`, path)
			return
		}
		assert.Equal(t, "/", path)
	})
}

func TestLocalIPs(t *testing.T) {
	t.Parallel()

	t.Run("leaves out loopback addresses", func(t *testing.T) {
		t.Parallel()

		ips := localIPs()

		for _, ip := range ips {
			parsed := net.ParseIP(ip)
			require.NotNil(t, parsed, ip)
			assert.False(t, parsed.IsLoopback(), "loopback addresses are left out")
		}
	})
}
