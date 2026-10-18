package service

import (
	"errors"
	"net"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/health"
)

func TestSystemService_Stats(t *testing.T) {
	t.Parallel()

	t.Run("returns the stats", func(t *testing.T) {
		t.Parallel()

		svc, _, _ := newTestSystemService(t, SystemOptions{})

		stats, err := svc.Stats(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "v1.2.3", stats.Version)
		assert.Positive(t, stats.MemTotal)
		assert.Positive(t, stats.DiskTotal)
		assert.GreaterOrEqual(t, stats.CPUPercent, 0.0)
		assert.LessOrEqual(t, stats.MemUsed, stats.MemTotal)
	})
}

func TestSystemService_Options(t *testing.T) {
	t.Parallel()

	t.Run("returns the configured commands", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		options := SystemOptions{
			ServerRestartCommand: f.LoremIpsumSentence(6),
			ServerStopCommand:    f.LoremIpsumSentence(6),
			KioskRestartCommand:  f.LoremIpsumSentence(6),
			KioskStopCommand:     f.LoremIpsumSentence(6),
			AllRestartCommand:    f.LoremIpsumSentence(6),
			AllStopCommand:       f.LoremIpsumSentence(6),
		}
		svc, _, _ := newTestSystemService(t, options)

		got := svc.Options()

		assert.Equal(t, options, got)
	})
}

func TestSystemService_ServerRestartEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "reports a configured command as enabled",
			command: apitest.Faker(t).LoremIpsumSentence(6),
			want:    true,
		},
		{
			name:    "reports an empty command as disabled",
			command: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, _ := newTestSystemService(t, SystemOptions{
				ServerRestartCommand: tt.command,
			})

			got := svc.ServerRestartEnabled()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSystemService_ServerStopEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "reports a configured command as enabled",
			command: apitest.Faker(t).LoremIpsumSentence(6),
			want:    true,
		},
		{
			name:    "reports an empty command as disabled",
			command: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, _ := newTestSystemService(t, SystemOptions{
				ServerStopCommand: tt.command,
			})

			got := svc.ServerStopEnabled()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSystemService_KioskRestartEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "reports a configured command as enabled",
			command: apitest.Faker(t).LoremIpsumSentence(6),
			want:    true,
		},
		{
			name:    "reports an empty command as disabled",
			command: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, _ := newTestSystemService(t, SystemOptions{
				KioskRestartCommand: tt.command,
			})

			got := svc.KioskRestartEnabled()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSystemService_KioskStopEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "reports a configured command as enabled",
			command: apitest.Faker(t).LoremIpsumSentence(6),
			want:    true,
		},
		{
			name:    "reports an empty command as disabled",
			command: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, _ := newTestSystemService(t, SystemOptions{
				KioskStopCommand: tt.command,
			})

			got := svc.KioskStopEnabled()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSystemService_AllRestartEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "reports a configured command as enabled",
			command: apitest.Faker(t).LoremIpsumSentence(6),
			want:    true,
		},
		{
			name:    "reports an empty command as disabled",
			command: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, _ := newTestSystemService(t, SystemOptions{
				AllRestartCommand: tt.command,
			})

			got := svc.AllRestartEnabled()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSystemService_AllStopEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "reports a configured command as enabled",
			command: apitest.Faker(t).LoremIpsumSentence(6),
			want:    true,
		},
		{
			name:    "reports an empty command as disabled",
			command: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, _ := newTestSystemService(t, SystemOptions{
				AllStopCommand: tt.command,
			})

			got := svc.AllStopEnabled()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSystemService_RestartServer(t *testing.T) {
	t.Parallel()

	t.Run("runs the configured command through a shell", func(t *testing.T) {
		t.Parallel()

		command := apitest.Faker(t).LoremIpsumSentence(6)
		svc, commands, _ := newTestSystemService(t, SystemOptions{
			ServerRestartCommand: command,
		})

		err := svc.RestartServer()

		require.NoError(t, err)
		assert.Equal(t, []commandCall{
			{
				Method: "Start",
				Name:   "sh",
				Args:   []string{"-c", command},
			},
		}, commands.Calls(), "the configured command runs exactly as written")
	})

	t.Run("returns the error of a failing start", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		svc, commands, _ := newTestSystemService(t, SystemOptions{
			ServerRestartCommand: f.LoremIpsumSentence(6),
		})
		commands.startErr = errors.New(f.LoremIpsumSentence(6))

		err := svc.RestartServer()

		assert.ErrorIs(t, err, commands.startErr)
	})
}

func TestSystemService_StopServer(t *testing.T) {
	t.Parallel()

	t.Run("runs the configured command through a shell", func(t *testing.T) {
		t.Parallel()

		command := apitest.Faker(t).LoremIpsumSentence(6)
		svc, commands, _ := newTestSystemService(t, SystemOptions{
			ServerStopCommand: command,
		})

		err := svc.StopServer()

		require.NoError(t, err)
		assert.Equal(t, []commandCall{
			{
				Method: "Start",
				Name:   "sh",
				Args:   []string{"-c", command},
			},
		}, commands.Calls(), "the configured command runs exactly as written")
	})

	t.Run("returns the error of a failing start", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		svc, commands, _ := newTestSystemService(t, SystemOptions{
			ServerStopCommand: f.LoremIpsumSentence(6),
		})
		commands.startErr = errors.New(f.LoremIpsumSentence(6))

		err := svc.StopServer()

		assert.ErrorIs(t, err, commands.startErr)
	})
}

func TestSystemService_RestartKiosk(t *testing.T) {
	t.Parallel()

	t.Run("runs the configured command through a shell", func(t *testing.T) {
		t.Parallel()

		command := apitest.Faker(t).LoremIpsumSentence(6)
		svc, commands, _ := newTestSystemService(t, SystemOptions{
			KioskRestartCommand: command,
		})

		err := svc.RestartKiosk()

		require.NoError(t, err)
		assert.Equal(t, []commandCall{
			{
				Method: "Start",
				Name:   "sh",
				Args:   []string{"-c", command},
			},
		}, commands.Calls(), "the configured command runs exactly as written")
	})

	t.Run("returns the error of a failing start", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		svc, commands, _ := newTestSystemService(t, SystemOptions{
			KioskRestartCommand: f.LoremIpsumSentence(6),
		})
		commands.startErr = errors.New(f.LoremIpsumSentence(6))

		err := svc.RestartKiosk()

		assert.ErrorIs(t, err, commands.startErr)
	})
}

func TestSystemService_StopKiosk(t *testing.T) {
	t.Parallel()

	t.Run("runs the configured command through a shell", func(t *testing.T) {
		t.Parallel()

		command := apitest.Faker(t).LoremIpsumSentence(6)
		svc, commands, _ := newTestSystemService(t, SystemOptions{
			KioskStopCommand: command,
		})

		err := svc.StopKiosk()

		require.NoError(t, err)
		assert.Equal(t, []commandCall{
			{
				Method: "Start",
				Name:   "sh",
				Args:   []string{"-c", command},
			},
		}, commands.Calls(), "the configured command runs exactly as written")
	})

	t.Run("returns the error of a failing start", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		svc, commands, _ := newTestSystemService(t, SystemOptions{
			KioskStopCommand: f.LoremIpsumSentence(6),
		})
		commands.startErr = errors.New(f.LoremIpsumSentence(6))

		err := svc.StopKiosk()

		assert.ErrorIs(t, err, commands.startErr)
	})
}

func TestSystemService_RestartAll(t *testing.T) {
	t.Parallel()

	t.Run("runs the configured command through a shell", func(t *testing.T) {
		t.Parallel()

		command := apitest.Faker(t).LoremIpsumSentence(6)
		svc, commands, _ := newTestSystemService(t, SystemOptions{
			AllRestartCommand: command,
		})

		err := svc.RestartAll()

		require.NoError(t, err)
		assert.Equal(t, []commandCall{
			{
				Method: "Start",
				Name:   "sh",
				Args:   []string{"-c", command},
			},
		}, commands.Calls(), "the configured command runs exactly as written")
	})

	t.Run("returns the error of a failing start", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		svc, commands, _ := newTestSystemService(t, SystemOptions{
			AllRestartCommand: f.LoremIpsumSentence(6),
		})
		commands.startErr = errors.New(f.LoremIpsumSentence(6))

		err := svc.RestartAll()

		assert.ErrorIs(t, err, commands.startErr)
	})
}

func TestSystemService_StopAll(t *testing.T) {
	t.Parallel()

	t.Run("runs the configured command through a shell", func(t *testing.T) {
		t.Parallel()

		command := apitest.Faker(t).LoremIpsumSentence(6)
		svc, commands, _ := newTestSystemService(t, SystemOptions{
			AllStopCommand: command,
		})

		err := svc.StopAll()

		require.NoError(t, err)
		assert.Equal(t, []commandCall{
			{
				Method: "Start",
				Name:   "sh",
				Args:   []string{"-c", command},
			},
		}, commands.Calls(), "the configured command runs exactly as written")
	})

	t.Run("returns the error of a failing start", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		svc, commands, _ := newTestSystemService(t, SystemOptions{
			AllStopCommand: f.LoremIpsumSentence(6),
		})
		commands.startErr = errors.New(f.LoremIpsumSentence(6))

		err := svc.StopAll()

		assert.ErrorIs(t, err, commands.startErr)
	})
}

func TestSystemService_ExitProcess(t *testing.T) {
	t.Parallel()

	t.Run("cancels the application context", func(t *testing.T) {
		t.Parallel()

		svc, _, canceled := newTestSystemService(t, SystemOptions{})

		svc.ExitProcess()

		assert.True(t, *canceled, "exiting cancels the application context")
	})
}

func TestSystemService_RebootHost(t *testing.T) {
	t.Parallel()

	t.Run("runs the reboot command", func(t *testing.T) {
		t.Parallel()

		svc, commands, _ := newTestSystemService(t, SystemOptions{})

		err := svc.RebootHost()

		require.NoError(t, err)
		reboot, rebootArgs := rebootCommand()
		calls := commands.Calls()
		require.Len(t, calls, 1)
		assert.Equal(t, reboot, calls[0].Name)
		assert.Equal(t, rebootArgs, calls[0].Args)
	})
}

func TestSystemService_ShutdownHost(t *testing.T) {
	t.Parallel()

	t.Run("runs the shutdown command", func(t *testing.T) {
		t.Parallel()

		svc, commands, _ := newTestSystemService(t, SystemOptions{})

		err := svc.ShutdownHost()

		require.NoError(t, err)
		shutdown, shutdownArgs := shutdownCommand()
		calls := commands.Calls()
		require.Len(t, calls, 1)
		assert.Equal(t, shutdown, calls[0].Name)
		assert.Equal(t, shutdownArgs, calls[0].Args)
	})
}

// newTestSystemService builds a system service on fake commands
// and reports through the flag whether it canceled the application.
func newTestSystemService(t *testing.T, options SystemOptions) (*SystemService, *fakeCommands, *bool) {
	t.Helper()

	commands := &fakeCommands{}
	canceled := false
	svc := NewSystemService(options, health.NewChecker("v1.2.3", nil), commands, func() { canceled = true })
	return svc, commands, &canceled
}

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
