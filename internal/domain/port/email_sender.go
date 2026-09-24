package port

import "context"

type EmailMessage struct {
	To       []string
	Subject  string
	BodyHTML string
}

type EmailService interface {
	// ProviderName returns the name of the configured email provider.
	ProviderName() string
	// Send delivers the supplied message to its recipients.
	Send(ctx context.Context, msg EmailMessage) error
}
