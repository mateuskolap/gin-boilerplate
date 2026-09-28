package mail

import (
	"context"
	"testing"

	"gin-boilerplate/config"
	"gin-boilerplate/internal/domain/port"
)

func TestSMTPServiceConfigurationAndMessageValidation(t *testing.T) {
	service, err := NewSMTPEmailService(&config.Config{SMTPHost: "localhost", SMTPPort: 2525, SMTPFrom: "sender@example.com"})
	if err != nil {
		t.Fatalf("NewSMTPEmailService() error = %v", err)
	}
	if service.ProviderName() != "smtp" {
		t.Fatalf("ProviderName() = %q", service.ProviderName())
	}
	if _, err := NewSMTPEmailService(&config.Config{SMTPHost: "localhost", SMTPPort: 2525, SMTPUser: "smtp-user", SMTPPassword: "secret", SMTPFrom: "sender@example.com"}); err != nil {
		t.Fatalf("NewSMTPEmailService() with authentication error = %v", err)
	}

	if err := service.Send(context.Background(), port.EmailMessage{To: []string{"not-an-email"}, Subject: "subject", BodyHTML: "<p>body</p>"}); err == nil {
		t.Fatal("Send() accepted an invalid recipient")
	}

	smtpService := service.(*smtpEmailService)
	smtpService.from = "not-an-email"
	if err := smtpService.Send(context.Background(), port.EmailMessage{To: []string{"recipient@example.com"}}); err == nil {
		t.Fatal("Send() accepted an invalid sender")
	}

	smtpService.from = "sender@example.com"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := smtpService.Send(ctx, port.EmailMessage{To: []string{"recipient@example.com"}}); err == nil {
		t.Fatal("Send() ignored cancellation before SMTP delivery")
	}
}
