package config

import (
	"testing"

	"github.com/barthjs/photopi/internal/mailer"
)

func TestValidateMail(t *testing.T) {
	t.Parallel()

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name:    "unknown driver",
				change:  func(c *Config) { c.Mail.Driver = "pigeon" },
				wantErr: "mail.driver",
			},
			{
				name:    "smtp without host",
				change:  func(c *Config) { c.Mail.Driver = mailer.DriverSMTP },
				wantErr: "MAIL_HOST",
			},
			{
				name:    "smtp with invalid port",
				change:  func(c *Config) { c.Mail.Driver = mailer.DriverSMTP; c.Mail.Host = "mail"; c.Mail.Port = 0 },
				wantErr: "mail.port",
			},
			{
				name:    "invalid reply-to",
				change:  func(c *Config) { c.Mail.ReplyTo = "not an address" },
				wantErr: "mail.reply_to",
			},
			{
				name:    "message id domain with at",
				change:  func(c *Config) { c.Mail.MessageIDDomain = "a@b" },
				wantErr: "mail.message_id_domain",
			},
		})
	})
}
