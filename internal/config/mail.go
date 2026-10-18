package config

import (
	"net/mail"
	"strings"

	"github.com/barthjs/photopi/internal/config/structconf"
	"github.com/barthjs/photopi/internal/mailer"
)

// MailConfig holds the mail settings.
type MailConfig struct {
	Driver          mailer.Driver `mapstructure:"driver" json:"driver" default:"log" values:"log,smtp" usage:"Mail driver, log writes every email to the log instead of sending it"`
	Host            string        `mapstructure:"host" json:"host" usage:"SMTP host of the smtp driver"`
	Port            int           `mapstructure:"port" json:"port" default:"587" usage:"SMTP port of the smtp driver"`
	Username        string        `mapstructure:"username" json:"username" usage:"SMTP username of the smtp driver"`
	Password        string        `mapstructure:"password" json:"password" secret:"true" usage:"SMTP password of the smtp driver"`
	FromAddress     string        `mapstructure:"from_address" json:"fromAddress" usage:"Sender email address"`
	FromName        string        `mapstructure:"from_name" json:"fromName" usage:"Sender display name"`
	ReplyTo         string        `mapstructure:"reply_to" json:"replyTo" usage:"Reply-To address of every email, such as a support address next to a no-reply sender, empty for none"`
	MessageIDDomain string        `mapstructure:"message_id_domain" json:"messageIdDomain" usage:"Domain of the Message-ID of every email, usually the sending domain, empty lets the SMTP client choose"`
}

// validateMail normalizes and checks the mail settings.
func validateMail(cfg *MailConfig) error {
	var p structconf.Problems

	if cfg.Driver == mailer.DriverSMTP {
		if cfg.Host == "" {
			p.Addf("mail driver smtp requires MAIL_HOST")
		}
		if cfg.Port < 1 || cfg.Port > 65535 {
			p.Addf("mail.port must be between 1 and 65535, got %d", cfg.Port)
		}
	}

	cfg.ReplyTo = strings.TrimSpace(cfg.ReplyTo)
	if cfg.ReplyTo != "" {
		if _, err := mail.ParseAddress(cfg.ReplyTo); err != nil {
			p.Addf("mail.reply_to must be an email address, got %q", cfg.ReplyTo)
		}
	}

	cfg.MessageIDDomain = strings.TrimSpace(cfg.MessageIDDomain)
	if strings.ContainsAny(cfg.MessageIDDomain, "@<> \t") {
		p.Addf("mail.message_id_domain must be a domain name, got %q", cfg.MessageIDDomain)
	}

	return p.Err()
}
