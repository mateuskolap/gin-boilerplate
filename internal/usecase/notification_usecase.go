package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"
	"slices"
	"strings"
	"uuid"
)

var allowedNotificationFilterFields = map[string]bool{
	"recipient":  true,
	"status":     true,
	"created_at": true,
	"id":         true,
}

type notificationUseCase struct {
	shared.BaseListUseCase[domain.Notification]
	shared.BaseFindUseCase[domain.Notification]
	notificationRepo domain.NotificationRepository
	emailService     port.EmailService
}

func NewNotificationUseCase(
	notificationRepo domain.NotificationRepository,
	emailService port.EmailService,
) domain.NotificationUseCase {
	return &notificationUseCase{
		BaseListUseCase: NewBaseListUseCase(
			notificationRepo,
			allowedNotificationFilterFields,
		),
		BaseFindUseCase: NewBaseFindUseCase(
			notificationRepo,
		),
		notificationRepo: notificationRepo,
		emailService:     emailService,
	}
}

func (n *notificationUseCase) SendEmail(ctx context.Context, userID *uuid.UUID, to []string, subject string, bodyHTML string) (*domain.Notification, error) {
	if len(to) == 0 || slices.Contains(to, "") {
		return nil, shared.NewAppError(
			shared.ErrTypeValidation,
			"Recipient email address is required",
			nil,
		)
	}

	notification := &domain.Notification{
		UserID:    userID,
		Channel:   domain.ChannelEmail,
		Provider:  n.emailService.ProviderName(),
		Recipient: strings.Join(to, ", "),
		Subject:   subject,
		Body:      bodyHTML,
		Status:    domain.NotificationStatusPending,
	}

	if err := n.notificationRepo.Create(ctx, notification); err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to create notification record",
			err,
		)
	}

	if err := n.emailService.Send(ctx, port.EmailMessage{
		To:       to,
		Subject:  subject,
		BodyHTML: bodyHTML,
	}); err != nil {
		errMsg := err.Error()
		notification.Status = domain.NotificationStatusFailed
		notification.ErrorMsg = &errMsg

		if err := n.notificationRepo.Update(ctx, notification); err != nil {
			return nil, shared.NewAppError(
				shared.ErrTypeInternal,
				"Failed to update notification status after email send failure",
				err,
			)
		}

		return notification, shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to send email",
			err,
		)
	}

	notification.Status = domain.NotificationStatusSent
	if err := n.notificationRepo.Update(ctx, notification); err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to update notification status after email send success",
			err,
		)
	}

	return notification, nil
}

func (n *notificationUseCase) EnqueueEmail(ctx context.Context, userID *uuid.UUID, to []string, subject string, bodyHTML string) (*domain.Notification, error) {
	panic("unimplemented")
}
