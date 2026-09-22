package mail

import (
	"context"
	"fmt"
	"gin-boilerplate/config"
	"gin-boilerplate/internal/domain/port"
	"net/smtp"
	"strconv"
	"strings"
)

type smtpEmailService struct {
	cfg *config.Config
}

func NewSMTPEmailService(cfg *config.Config) port.EmailService {
	return &smtpEmailService{
		cfg: cfg,
	}
}

func (s *smtpEmailService) ProviderName() string {
	return "smtp"
}

func (s *smtpEmailService) Send(ctx context.Context, msg port.EmailMessage) error {
	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, strconv.Itoa(s.cfg.SMTPPort))
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)

	toHeader := strings.Join(msg.To, ", ")

	message := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: %s\r\n\r\n"+
		"%s",
		s.cfg.SMTPFrom,
		toHeader,
		msg.Subject,
		"text/html; charset=UTF-8",
		msg.BodyHTML,
	)

	return smtp.SendMail(addr, auth, s.cfg.SMTPUser, msg.To, []byte(message))
}
