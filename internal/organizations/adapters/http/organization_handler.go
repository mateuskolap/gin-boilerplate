package http

import (
	"gin-boilerplate/internal/delivery/http/common"
	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/response"
	"gin-boilerplate/internal/domain/shared"
	organizationdto "gin-boilerplate/internal/organizations/adapters/http/dto"
	organizationdomain "gin-boilerplate/internal/organizations/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type OrganizationHandler struct {
	organizationUseCase organizationdomain.OrganizationUseCase
}

func NewOrganizationHandler(organizationUseCase organizationdomain.OrganizationUseCase) *OrganizationHandler {
	return &OrganizationHandler{
		organizationUseCase: organizationUseCase,
	}
}

func (h *OrganizationHandler) FindOrganization(c *gin.Context) {
	organizationID, err := common.ParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	organization, err := h.organizationUseCase.Find(c.Request.Context(), organizationID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Organization retrieved successfully", organizationdto.ToOrganizationResponse(organization))
}

func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	params, err := common.PaginationParams(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var filters []shared.Filter
	if name := c.Query("name"); name != "" {
		filters = append(filters, shared.Filter{
			Field:    "name",
			Operator: shared.OperatorLike,
			Value:    "%" + name + "%",
		})
	}

	result, err := h.organizationUseCase.List(c.Request.Context(), params, filters)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := dto.ToPaginatedResponse(result, organizationdto.ToOrganizationResponse)
	response.Success(c, http.StatusOK, "Organizations retrieved successfully", resp)
}

func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	req, err := common.BindJSON[organizationdto.CreateOrganizationRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	organization := &organizationdomain.Organization{
		Name: req.Name,
	}

	if err := h.organizationUseCase.Create(c.Request.Context(), organization); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusCreated, "Organization created successfully", organizationdto.ToOrganizationResponse(organization))
}

func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	organizationID, err := common.ParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	req, err := common.BindJSON[organizationdto.UpdateOrganizationRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	organization := &organizationdomain.Organization{
		ID:   organizationID,
		Name: req.Name,
	}

	if err := h.organizationUseCase.Update(c.Request.Context(), organization); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Organization updated successfully", organizationdto.ToOrganizationResponse(organization))
}

func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	organizationID, err := common.ParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.organizationUseCase.Delete(c.Request.Context(), organizationID); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
