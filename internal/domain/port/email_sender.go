package port

import "context"

type EmailMessage struct {
	To       []string
	Subject  string
	BodyHTML string
}

type EmailService interface {
	ProviderName() string
	Send(ctx context.Context, msg EmailMessage) error
}
