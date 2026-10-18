package service

import (
	"context"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/barthjs/photopi/internal/health"
)

// SystemService reports the host state and runs the configured system commands.
type SystemService struct {
	options  SystemOptions
	health   *health.Checker
	commands CommandRunner
	cancel   context.CancelFunc
}

// SystemOptions holds the shell commands behind the actions of the system page. An empty command disables its action.
type SystemOptions struct {
	ServerRestartCommand string
	ServerStopCommand    string
	KioskRestartCommand  string
	KioskStopCommand     string
	AllRestartCommand    string
	AllStopCommand       string
}

// NewSystemService creates a SystemService. ExitProcess shuts down the process through cancel.
func NewSystemService(options SystemOptions, healthChecker *health.Checker, commands CommandRunner, cancel context.CancelFunc) *SystemService {
	return &SystemService{
		options:  options,
		health:   healthChecker,
		commands: commands,
		cancel:   cancel,
	}
}

// Stats is the state of the host and the application.
type Stats struct {
	Version    string
	Uptime     time.Duration
	Hostname   string
	IPs        []string
	CPUPercent float64
	MemUsed    uint64
	MemTotal   uint64
	DiskUsed   uint64
	DiskTotal  uint64
}

// Stats gathers a fresh snapshot of host and process metrics.
func (s *SystemService) Stats(ctx context.Context) (Stats, error) {
	hostname, _ := os.Hostname()

	cpuPercents, err := cpu.PercentWithContext(ctx, 200*time.Millisecond, false)
	if err != nil {
		return Stats{}, err
	}
	var cpuPercent float64
	if len(cpuPercents) > 0 {
		cpuPercent = cpuPercents[0]
	}

	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return Stats{}, err
	}

	du, err := disk.UsageWithContext(ctx, diskUsagePath())
	if err != nil {
		return Stats{}, err
	}

	return Stats{
		Version:    s.health.Version(),
		Uptime:     s.health.Uptime(),
		Hostname:   hostname,
		IPs:        localIPs(),
		CPUPercent: cpuPercent,
		MemUsed:    vm.Used,
		MemTotal:   vm.Total,
		DiskUsed:   du.Used,
		DiskTotal:  du.Total,
	}, nil
}

// Options returns the configured commands of the system actions, an empty one disables its action.
func (s *SystemService) Options() SystemOptions {
	return s.options
}

// ServerRestartEnabled reports whether a command to restart the application's own service is configured.
func (s *SystemService) ServerRestartEnabled() bool { return s.options.ServerRestartCommand != "" }

// ServerStopEnabled reports whether a command to stop the application's own service is configured.
func (s *SystemService) ServerStopEnabled() bool { return s.options.ServerStopCommand != "" }

// KioskRestartEnabled reports whether a command to restart the kiosk browser's service is configured.
func (s *SystemService) KioskRestartEnabled() bool { return s.options.KioskRestartCommand != "" }

// KioskStopEnabled reports whether a command to stop the kiosk browser's service is configured.
func (s *SystemService) KioskStopEnabled() bool { return s.options.KioskStopCommand != "" }

// AllRestartEnabled reports whether a command to restart both services at once is configured.
func (s *SystemService) AllRestartEnabled() bool { return s.options.AllRestartCommand != "" }

// AllStopEnabled reports whether a command to stop both services at once is configured.
func (s *SystemService) AllStopEnabled() bool { return s.options.AllStopCommand != "" }

// RestartServer runs the configured command to restart the application's own service.
func (s *SystemService) RestartServer() error { return s.runCommand(s.options.ServerRestartCommand) }

// StopServer runs the configured command to stop the application's own service.
func (s *SystemService) StopServer() error { return s.runCommand(s.options.ServerStopCommand) }

// RestartKiosk runs the configured command to restart the kiosk browser's service.
func (s *SystemService) RestartKiosk() error { return s.runCommand(s.options.KioskRestartCommand) }

// StopKiosk runs the configured command to stop the kiosk browser's service.
func (s *SystemService) StopKiosk() error { return s.runCommand(s.options.KioskStopCommand) }

// RestartAll runs the configured command to restart both services at once.
func (s *SystemService) RestartAll() error { return s.runCommand(s.options.AllRestartCommand) }

// StopAll runs the configured command to stop both services at once.
func (s *SystemService) StopAll() error { return s.runCommand(s.options.AllStopCommand) }

// runCommand starts command via a shell without waiting for it and only reports whether starting it failed.
func (s *SystemService) runCommand(command string) error {
	return s.commands.Start("sh", "-c", command)
}

// ExitProcess gracefully shuts down the application process without a configured command.
// Whether it comes back up depends on its supervisor.
func (s *SystemService) ExitProcess() {
	s.cancel()
}

// RebootHost reboots the host machine.
func (s *SystemService) RebootHost() error {
	name, args := rebootCommand()
	return s.commands.Start(name, args...)
}

// ShutdownHost powers off the host machine.
func (s *SystemService) ShutdownHost() error {
	name, args := shutdownCommand()
	return s.commands.Start(name, args...)
}

// rebootCommand returns the OS command that reboots the host machine.
func rebootCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "shutdown", []string{"/r", "/t", "0"}
	}
	return "reboot", nil
}

// shutdownCommand returns the OS command that powers off the host machine.
func shutdownCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "shutdown", []string{"/s", "/t", "0"}
	}
	return "shutdown", []string{"-h", "now"}
}

// diskUsagePath is the filesystem root reported as disk usage.
func diskUsagePath() string {
	if runtime.GOOS == "windows" {
		return `C:\`
	}
	return "/"
}

// localIPs returns every IP address bound to a local network interface except the loopback addresses.
func localIPs() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}

	var ips []string
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		ips = append(ips, ipNet.IP.String())
	}

	return ips
}
