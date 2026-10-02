package dto

import (
	"gin-boilerplate/internal/delivery/http/dto"
	organizationdomain "gin-boilerplate/internal/organizations/domain"
	"time"
	"uuid"
)

type OrganizationResponse struct {
	ID        uuid.UUID `json:"id" example:"a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"`
	Name      string    `json:"name" example:"My Organization"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-01T00:00:00Z"`
}

type PaginatedOrganizationResponse = dto.PaginatedResponse[OrganizationResponse]

type CreateOrganizationRequest struct {
	Name string `json:"name" binding:"required,min=2,max=255" example:"My Organization"`
}

type UpdateOrganizationRequest struct {
	Name string `json:"name" binding:"required,min=2,max=255" example:"My Organization Updated"`
}

func ToOrganizationResponse(organization *organizationdomain.Organization) OrganizationResponse {
	return OrganizationResponse{
		ID:        organization.ID,
		Name:      organization.Name,
		CreatedAt: organization.CreatedAt,
		UpdatedAt: organization.UpdatedAt,
	}
}
