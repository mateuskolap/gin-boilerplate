package domain

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
	"time"
	"uuid"
)

type NotificationStatus string
type NotificationChannel string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"

	ChannelEmail NotificationChannel = "email"
)

type Notification struct {
	shared.BaseModel
	UserID    *uuid.UUID          `json:"user_id,omitempty" gorm:"type:uuid;index"`
	Channel   NotificationChannel `json:"channel" gorm:"type:varchar(20);not null"`
	Provider  string              `json:"provider" gorm:"type:varchar(50);not null"`
	Recipient string              `json:"recipient" gorm:"type:varchar(255);not null;index"`
	Subject   string              `json:"subject" gorm:"type:varchar(255);not null"`
	Body      string              `json:"body" gorm:"type:text;not null"`
	Status    NotificationStatus  `json:"status" gorm:"type:varchar(20);not null;default:'pending';index"`
	ErrorMsg  *string             `json:"error_msg,omitempty" gorm:"type:text"`
	SentAt    *time.Time          `json:"sent_at,omitempty"`
}

type NotificationRepository interface {
	shared.BaseRepository[Notification]
}

type NotificationUseCase interface {
	shared.BaseListUseCase[Notification]
	shared.BaseFindUseCase[Notification]

	SendEmail(ctx context.Context, userID *uuid.UUID, to []string, subject string, bodyHTML string) (*Notification, error)
	EnqueueEmail(ctx context.Context, userID *uuid.UUID, to []string, subject string, bodyHTML string) (*Notification, error)
}
