package mail

import (
	"context"
	"fmt"
	"strings"

	"gin-boilerplate/config"
	"gin-boilerplate/internal/domain/port"

	mail "github.com/wneessen/go-mail"
)

type smtpEmailService struct {
	client *mail.Client
	from   string
}

func NewSMTPEmailService(cfg *config.Config) (port.EmailService, error) {
	options := []mail.Option{
		mail.WithPort(cfg.SMTPPort),
		mail.WithTLSPolicy(mail.TLSMandatory),
	}
	if strings.TrimSpace(cfg.SMTPUser) != "" {
		options = append(options,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(cfg.SMTPUser),
			mail.WithPassword(cfg.SMTPPassword),
		)
	}

	client, err := mail.NewClient(cfg.SMTPHost, options...)
	if err != nil {
		return nil, fmt.Errorf("configure SMTP client: %w", err)
	}

	return &smtpEmailService{client: client, from: cfg.SMTPFrom}, nil
}

func (s *smtpEmailService) ProviderName() string {
	return "smtp"
}

func (s *smtpEmailService) Send(ctx context.Context, message port.EmailMessage) error {
	msg := mail.NewMsg()
	if err := msg.From(s.from); err != nil {
		return fmt.Errorf("set email sender: %w", err)
	}
	if err := msg.To(message.To...); err != nil {
		return fmt.Errorf("set email recipients: %w", err)
	}
	msg.Subject(message.Subject)
	msg.SetBodyString(mail.TypeTextHTML, message.BodyHTML)

	if err := s.client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("send email through SMTP: %w", err)
	}
	return nil
}
