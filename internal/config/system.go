package config

import (
	"strings"

	"github.com/barthjs/photopi/internal/config/structconf"
)

// SystemConfig holds the system settings, the shell commands behind the system actions of the admin panel.
type SystemConfig struct {
	ServerRestartCommand string `mapstructure:"server_restart_command" json:"serverRestartCommand" default:"systemctl --user restart photopi.service" usage:"Shell command that restarts the service of the application, empty disables the button"`
	ServerStopCommand    string `mapstructure:"server_stop_command" json:"serverStopCommand" default:"systemctl --user stop photopi.service" usage:"Shell command that stops the service of the application, empty disables the button"`
	KioskRestartCommand  string `mapstructure:"kiosk_restart_command" json:"kioskRestartCommand" default:"systemctl --user restart photopi-kiosk.service" usage:"Shell command that restarts the service of the kiosk browser, empty disables the button"`
	KioskStopCommand     string `mapstructure:"kiosk_stop_command" json:"kioskStopCommand" default:"systemctl --user stop photopi-kiosk.service" usage:"Shell command that stops the service of the kiosk browser, empty disables the button"`
	AllRestartCommand    string `mapstructure:"all_restart_command" json:"allRestartCommand" default:"systemctl --user restart photopi.service photopi-kiosk.service" usage:"Shell command that restarts the application and the kiosk browser in one step, empty disables the button"`
	AllStopCommand       string `mapstructure:"all_stop_command" json:"allStopCommand" default:"systemctl --user stop photopi-kiosk.service photopi.service" usage:"Shell command that stops the application and the kiosk browser in one step, empty disables the button"`
}

// validateSystem normalizes and checks the system settings.
func validateSystem(cfg *SystemConfig) error {
	var p structconf.Problems

	commands := map[string]*string{
		"system.server_restart_command": &cfg.ServerRestartCommand,
		"system.server_stop_command":    &cfg.ServerStopCommand,
		"system.kiosk_restart_command":  &cfg.KioskRestartCommand,
		"system.kiosk_stop_command":     &cfg.KioskStopCommand,
		"system.all_restart_command":    &cfg.AllRestartCommand,
		"system.all_stop_command":       &cfg.AllStopCommand,
	}
	for key, command := range commands {
		*command = strings.TrimSpace(*command)
		if len(*command) > maxPath {
			p.Addf("%s must be at most %d characters", key, maxPath)
		}
	}

	return p.Err()
}
